package e2eutil

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"text/template"
	"time"

	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/util"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	managerName = "porter-manager"
)

type DeployFunc func() error
type TestFunc func()
type TestCase struct {
	Name           string
	ControllerAS   int
	ControllerIP   string
	ControllerPort int
	ControllerName string

	RouterIP               string
	RouterAS               int
	UsePortForward         bool
	RouterConfigPath       string
	ControllerConfigPath   string
	IsPassiveMode          bool
	RouterTemplatePath     string
	ControllerTemplatePath string
	Namespace              string
	K8sClient              client.Client
	KustomizePath          string
	routeContainerID       string
	DeployYamlPath         string
	stopRouter             chan struct{}
	isLocal                bool
	TestDeploymentName     string

	InjectTest TestFunc
}

var routerContainerNotExist = fmt.Errorf("containerid is empty")

func (t *TestCase) WaitForControllerUp() error {
	return WaitForController(t.K8sClient, t.Namespace, managerName, 5*time.Second, 3*time.Minute)
}

func (t *TestCase) SetRouterContainerID(id string) {
	t.routeContainerID = id
}

func (t *TestCase) WriteConfig(temppath, output string) error {
	temp, err := template.ParseFiles(temppath)
	if err != nil {
		log.Println("Error in parsing template: " + temppath)
		return err
	}
	w, err := os.Create(output)
	if err != nil {
		return err
	}
	return temp.Execute(w, t)
}

func (t *TestCase) GenerateControllerConfig() (string, error) {
	temp, err := template.ParseFiles(t.ControllerTemplatePath)
	if err != nil {
		log.Println("Error in parsing template: " + t.ControllerTemplatePath)
		return "", err
	}
	sb := new(bytes.Buffer)
	err = temp.Execute(sb, t)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

func (t *TestCase) CheckNetwork() {
	ip := util.GetOutboundIP()
	if ip == t.RouterIP {
		t.isLocal = true
	}
	jenkinsIP := os.Getenv("JENKINS_IP")
	if jenkinsIP == t.RouterIP {
		t.isLocal = true
	}
}

func (t *TestCase) IsLocal() bool {
	return t.isLocal
}

func (t *TestCase) StartRemoteRoute() error {
	//route config
	t.CheckNetwork()
	routeGeneratedConfig := "/tmp/route.toml"
	err := t.WriteConfig(t.RouterTemplatePath, routeGeneratedConfig)
	if err != nil {
		return err
	}
	if !t.isLocal {
		err = ScpFileToRemote(routeGeneratedConfig, t.RouterConfigPath, t.RouterIP)
		if err != nil {
			log.Printf("Error in transfer router config, error: %s", err.Error())
			return err
		}
	}
	//start a container this will block until container end
	id, err := RunGoBGPContainer(t.RouterConfigPath)
	if err != nil {
		log.Println("Failed to start gobgp container")
		return err
	}
	t.routeContainerID = id
	return nil
}

func (t *TestCase) StopRouter() error {
	if t.routeContainerID == "" {
		return routerContainerNotExist
	}
	return StopGoBGPContainer(t.routeContainerID)
}

func (t *TestCase) ReplaceConfigMap() error {
	data, err := t.GenerateControllerConfig()
	if err != nil {
		return err
	}
	return wait.Poll(time.Second, time.Second*10, func() (done bool, err error) {
		config := &corev1.ConfigMap{}
		err = t.K8sClient.Get(context.TODO(), types.NamespacedName{Namespace: t.Namespace, Name: "bgp-cfg"}, config)
		if err != nil {
			log.Println(err.Error())
			return
		}
		config.Data = map[string]string{"config.toml": data}
		err = t.K8sClient.Update(context.TODO(), config)
		if err != nil {
			log.Println(err.Error())
			return
		}
		return true, nil
	})
}

func (t *TestCase) DeployYaml() error {
	err := KubectlApply(t.DeployYamlPath)
	if err != nil {
		log.Println("kubectl apply failed")
		return err
	}
	err = t.ReplaceConfigMap()
	if err != nil {
		log.Println("Failed to replace configmap")
	}
	err = t.WaitForControllerUp()
	if err != nil {
		log.Println("timeout waiting for controller up")
		return err
	}
	log.Println("Controller is up now")
	return nil
}

func (t *TestCase) GetRouterLog() (string, error) {
	if t.routeContainerID == "" {
		return "", routerContainerNotExist
	}
	return GetContainerLog(t.routeContainerID)
}

// Change configure of TestCase to test some behabiour
func (t *TestCase) StartDefaultTest(workspace string) {
	service := &corev1.Service{}
	Expect(t.StartRemoteRoute()).NotTo(HaveOccurred(), "Error in starting remote bgp")
	defer t.StopRouter()
	//apply yaml
	Expect(t.DeployYaml()).ShouldNot(HaveOccurred(), "Failed to deploy yaml")
	defer func() {
		Expect(t.DeleteController()).ShouldNot(HaveOccurred(), "Failed to delete controller")
	}()

	//testing
	eip := &networkv1alpha1.Eip{}
	eip.Name = "test-eip"
	eip.Spec.Address = "1.1.1.0/24"
	Expect(t.K8sClient.Create(context.TODO(), eip)).NotTo(HaveOccurred())
	defer func() {
		t.K8sClient.Delete(context.TODO(), eip)
		WaitForDeletion(t.K8sClient, eip, 5*time.Second, 1*time.Minute)
	}()

	//apply service
	serviceStr := `{
		"kind": "Service",
		"apiVersion": "v1",
		"metadata": {
			"name": "xxx",
			"annotations": {
				"lb.kubesphere.io/v1alpha1": "porter"
			}
		},
		"spec": {
			"selector": {
				"app": "test-app"
			},
			"type": "LoadBalancer",
			"ports": [
				{
					"name": "http",
					"port": 8088,
					"targetPort": 80
				}
			]
		}
	}`

	reader := strings.NewReader(serviceStr)
	Expect(yaml.NewYAMLOrJSONDecoder(reader, 10).Decode(service)).ShouldNot(HaveOccurred())
	service.Name = t.TestDeploymentName
	service.Namespace = t.Namespace
	serviceType := types.NamespacedName{
		Namespace: t.Namespace,
		Name:      service.Name,
	}
	Expect(t.K8sClient.Create(context.TODO(), service)).ShouldNot(HaveOccurred())
	Eventually(func() error {
		err := t.K8sClient.Get(context.TODO(), serviceType, service)
		return err
	}, time.Second*30, 5*time.Second).Should(Succeed())
	defer func() {
		Expect(t.DeleteServiceGracefully(service)).ShouldNot(HaveOccurred())
	}()

	//Service should get its eip
	Eventually(func() error {
		err := t.K8sClient.Get(context.TODO(), serviceType, service)
		if err != nil {
			return err
		}
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			_, ipnet, _ := net.ParseCIDR(eip.Spec.Address)
			if ipnet.Contains(net.ParseIP(service.Status.LoadBalancer.Ingress[0].IP)) {
				return nil
			}
		}
		return fmt.Errorf("Failed to get correct ingress")
	}, 2*time.Minute, time.Second).Should(Succeed())
	//check route in bird
	Eventually(func() error {
		ips, err := GetServiceNodesIP(t.K8sClient, serviceType)
		if err != nil {
			return err
		}
		if len(ips) < 2 {
			return fmt.Errorf("Service Not Ready")
		}
		s, err := t.CheckBGPRoute()
		if err != nil {
			log.Printf("current mode: %v,err: %s", t.IsLocal(), s)
			return err
		}
		for _, ip := range ips {
			if !strings.Contains(s, ip) {
				return fmt.Errorf("No routes in GoBGP")
			}
		}
		return nil
	}, time.Minute, 5*time.Second).Should(Succeed())
	//inject test

	if t.InjectTest != nil {
		t.InjectTest()
	}
	//CheckLog
	log, err := t.GetRouterLog()
	Expect(err).ShouldNot(HaveOccurred(), "Failed to get log of router")
	Expect(log).ShouldNot(ContainSubstring("error"))

	podlist := &corev1.PodList{}
	Expect(t.K8sClient.List(context.TODO(), podlist, client.InNamespace(t.Namespace), client.MatchingLabels{"app": "porter-manager"})).ShouldNot(HaveOccurred())
	managerPodName := podlist.Items[0].Name
	log, err = CheckManagerLog(t.Namespace, managerPodName, fmt.Sprintf("%s/test/manager_%s.porterlog", workspace, t.Name))
	Expect(err).ShouldNot(HaveOccurred(), log)
	log, err = CheckAgentLog(t.Namespace, "porter-agent", fmt.Sprintf("%s/test/agent_%s", workspace, t.Name), t.K8sClient)
	Expect(err).ShouldNot(HaveOccurred(), log)
}

func (t *TestCase) DeleteServiceGracefully(service *corev1.Service) error {
	return WaitForDeletion(t.K8sClient, service, time.Second*5, time.Minute)
}

func (t *TestCase) CheckBGPRoute() (string, error) {
	if t.isLocal {
		return checkBGPRoute(true)
	} else {
		return checkBGPRoute(false, t.RouterIP)
	}
}

func (t *TestCase) DeleteController() error {
	deploy := &appsv1.Deployment{}
	err := t.K8sClient.Get(context.TODO(), types.NamespacedName{Name: t.ControllerName, Namespace: t.Namespace}, deploy)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return WaitForDeletion(t.K8sClient, deploy, time.Second*5, time.Minute)
}
