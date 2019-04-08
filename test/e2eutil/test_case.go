package e2eutil

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	networkv1alpha1 "github.com/kubesphere/porter/pkg/apis/network/v1alpha1"
	"github.com/kubesphere/porter/pkg/util"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	managerPodName = "controller-manager-0"
	managerName    = "controller-manager"
)

type DeployFunc func() error
type TestFunc func()
type TestCase struct {
	ControllerAS   int
	ControllerIP   string
	ControllerPort int

	RouterIP               string
	RouterAS               int
	UsePortForward         bool
	RouterConfigPath       string
	KustomizeConfigPath    string
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
}

var routerContainerNotExist = fmt.Errorf("containerid is empty")

func (t *TestCase) WaitForControllerUp() error {
	return WaitForController(t.K8sClient, t.Namespace, managerName, 5*time.Second, 2*time.Minute)
}

func (t *TestCase) SetRouterContainerID(id string) {
	t.routeContainerID = id
}
func WriteConfig(temppath, output string, t *TestCase) error {
	temp, err := template.ParseFiles(temppath)
	if err != nil {
		log.Println("Error in parsing template: " + temppath)
		return err
	}
	f, err := os.Create(output)
	if err != nil {
		log.Println("Error in writing template: " + temp.Name())
		return err
	}
	return temp.Execute(f, t)
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
	err := WriteConfig(t.RouterTemplatePath, routeGeneratedConfig, t)
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
func (t *TestCase) DeployYaml() error {
	//generate config
	err := WriteConfig(t.ControllerTemplatePath, t.KustomizeConfigPath, t)
	if err != nil {
		log.Println("Failed to generate controller config")
		return err
	}
	//kustomize
	err = KustomizeBuild(t.KustomizePath, t.DeployYamlPath)
	if err != nil {
		log.Println("Kustomize failed")
		return err
	}
	err = KubectlApply(t.DeployYamlPath)
	if err != nil {
		log.Println("kubectl apply failed")
		return err
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
	serviceTypes := types.NamespacedName{Namespace: "default", Name: "mylbapp"}
	Expect(t.StartRemoteRoute()).NotTo(HaveOccurred(), "Error in starting remote bgp")
	defer t.StopRouter()
	//apply yaml
	Expect(t.DeployYaml()).ShouldNot(HaveOccurred(), "Failed to deploy yaml")
	defer func() {
		Expect(KubectlDelete(t.DeployYamlPath)).ShouldNot(HaveOccurred(), "Failed to delete yaml")
	}()

	//testing
	eip := &networkv1alpha1.EIP{}
	reader, err := os.Open(workspace + "/config/samples/network_v1alpha1_eip.yaml")
	Expect(err).NotTo(HaveOccurred(), "Cannot read sample yamls")
	err = yaml.NewYAMLOrJSONDecoder(reader, 10).Decode(eip)
	Expect(err).NotTo(HaveOccurred(), "Cannot unmarshal yamls")
	if eip.Namespace == "" {
		eip.Namespace = "default"
	}
	err = t.K8sClient.Create(context.TODO(), eip)
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		t.K8sClient.Delete(context.TODO(), eip)
		WaitForDeletion(t.K8sClient, eip, 5*time.Second, 1*time.Minute)
	}()

	//apply service
	service1Path := workspace + "/config/samples/service.yaml"
	Expect(KubectlApply(service1Path)).ShouldNot(HaveOccurred())
	service := &corev1.Service{}
	Eventually(func() error {
		err := t.K8sClient.Get(context.TODO(), serviceTypes, service)
		return err
	}, time.Second*30, 5*time.Second).Should(Succeed())
	defer t.DeleteServiceGracefully(service, service1Path)

	//Service should get its eip
	Eventually(func() error {
		service := &corev1.Service{}
		err := t.K8sClient.Get(context.TODO(), serviceTypes, service)
		if err != nil {
			return err
		}
		if len(service.Status.LoadBalancer.Ingress) > 0 && service.Status.LoadBalancer.Ingress[0].IP == eip.Spec.Address {
			return nil
		}
		return fmt.Errorf("Failed")
	}, 2*time.Minute, time.Second).Should(Succeed())
	//check route in bird
	if t.IsLocal() {
		Eventually(func() error {
			s, err := CheckBGPRoute()
			if err != nil {
				return err
			}
			ips, err := GetServiceNodesIP(t.K8sClient, serviceTypes.Namespace, serviceTypes.Name)
			if err != nil {
				return err
			}
			if len(ips) < 2 {
				return fmt.Errorf("Service Not Ready")
			}
			for _, ip := range ips {
				if !strings.Contains(s, ip) {
					return fmt.Errorf("No routes in GoBGP")
				}
			}
			return nil
		}, time.Minute, 5*time.Second).Should(Succeed())
	} else {
		session, err := QuickConnectUsingDefaultSSHKey(t.RouterIP)
		Expect(err).NotTo(HaveOccurred(), "Connect Bird using private key FAILED")
		defer session.Close()
		stdinBuf, err := session.StdinPipe()
		var outbt, errbt bytes.Buffer
		session.Stdout = &outbt
		session.Stderr = &errbt
		err = session.Shell()
		Expect(err).ShouldNot(HaveOccurred(), "Failed to start ssh shell")
		Eventually(func() error {
			stdinBuf.Write([]byte("gobgp global rib\n"))
			ips, err := GetServiceNodesIP(t.K8sClient, serviceTypes.Namespace, serviceTypes.Name)
			if err != nil {
				return err
			}
			s := outbt.String() + errbt.String()
			for _, ip := range ips {
				if !strings.Contains(s, ip) {
					return fmt.Errorf("No routes in GoBGP")
				}
			}
			return nil
		}, time.Minute, 5*time.Second).Should(Succeed())
	}
	//CheckLog
	log, err := t.GetRouterLog()
	Expect(err).ShouldNot(HaveOccurred(), "Failed to get log of router")
	Expect(log).ShouldNot(ContainSubstring("error"))

	log, err = CheckManagerLog(t.Namespace, managerName)
	Expect(err).ShouldNot(HaveOccurred(), log)
	log, err = CheckAgentLog(t.Namespace, "porter-agent", t.K8sClient)
	Expect(err).ShouldNot(HaveOccurred(), log)
}

func (t *TestCase) DeleteServiceGracefully(service *corev1.Service, yaml string) {
	Expect(KubectlDelete(yaml)).ShouldNot(HaveOccurred())
	Expect(WaitForDeletion(t.K8sClient, service, time.Second*5, time.Minute)).ShouldNot(HaveOccurred(), "Failed waiting for services deletion")
}
