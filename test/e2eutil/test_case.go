package e2eutil

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	"log"
	"net"
	"os"
	"strings"
	"text/template"
	"time"

	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	bgpserver "github.com/kubesphere/porter/pkg/bgp/serverd"
	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/util"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	managerName = "porter-manager"
)

type DeployFunc func() error
type TestFunc func()
type TestCase struct {
	ControllerAS   int
	ControllerIP   string
	ControllerPort int

	RouterIP           string
	RouterAS           int
	RouterPort         int
	RouterConfigPath   string //for bgp docker config
	RouterTemplatePath string
	routeContainerID   string
	isLocal            bool

	//Neighbor
	UsePortForward bool
	IsPassiveMode  bool

	Namespace string
	K8sClient client.Client

	Layer2 bool

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
	routeGeneratedConfig := GoBgpConfig
	err := t.WriteConfig(t.RouterTemplatePath, routeGeneratedConfig)
	if err != nil {
		return err
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

func (t *TestCase) GetRouterLog() (string, error) {
	if t.routeContainerID == "" {
		return "", routerContainerNotExist
	}
	return GetContainerLog(t.routeContainerID)
}

func (t *TestCase) prepareDeployment() (*appsv1.Deployment, error) {
	deployStr := `{
	"kind": "Deployment",
	"apiVersion": "apps/v1",
	"metadata": {
		"name": "test-app",
		"creationTimestamp": null,
		"labels": {
			"app": "test-app"
		}
	},
	"spec": {
		"replicas": 3,
		"selector": {
			"matchLabels": {
				"app": "test-app"
			}
		},
		"template": {
			"metadata": {
				"labels": {
					"app": "test-app"
				}
			},
			"spec": {
				"containers": [
					{
						"name": "nginx",
						"image": "nginx:alpine"
					}
				]
			}
		}
	}
}`
	nginxDeploy := new(appsv1.Deployment)
	reader := strings.NewReader(deployStr)
	err := yaml.NewYAMLOrJSONDecoder(reader, 10).Decode(nginxDeploy)
	if err != nil {
		return nginxDeploy, err
	}
	nginxDeploy.Namespace = t.Namespace

	return nginxDeploy, t.K8sClient.Create(context.TODO(), nginxDeploy)
}

// Change configure of TestCase to test some behabiour
func (t *TestCase) StartDefaultTest(workspace string) {
	service := &corev1.Service{}
	Expect(t.StartRemoteRoute()).NotTo(HaveOccurred(), "Error in starting remote bgp")
	defer t.StopRouter()

	bgpPeer := &networkv1alpha1.BgpPeer{
		Spec: bgpserver.BgpPeerSpec{
			Config: bgpserver.NeighborConfig{
				PeerAs:          uint32(t.RouterAS),
				NeighborAddress: t.RouterIP,
			},
			AddPaths:         bgpserver.AddPaths{},
			Transport:        bgpserver.Transport{},
			UsingPortForward: true,
		},
	}
	bgpPeer.Name = "test-peer"
	Expect(t.K8sClient.Create(context.TODO(), bgpPeer)).NotTo(HaveOccurred())
	defer func() {
		t.K8sClient.Delete(context.TODO(), bgpPeer)
		WaitForDeletion(t.K8sClient, bgpPeer, 5*time.Second, 1*time.Minute)
	}()

	bgpConf := &networkv1alpha1.BgpConf{
		Spec: bgpserver.BgpConfSpec{
			Port:     int32(t.ControllerPort),
			As:       uint32(t.ControllerAS),
			RouterId: t.ControllerIP,
		},
	}
	bgpConf.Name = "test-bgpconf"
	Expect(t.K8sClient.Create(context.TODO(), bgpConf)).NotTo(HaveOccurred())
	defer func() {
		t.K8sClient.Delete(context.TODO(), bgpConf)
		WaitForDeletion(t.K8sClient, bgpConf, 5*time.Second, 1*time.Minute)
	}()

	//testing
	eip := &networkv1alpha1.Eip{}
	eip.Name = "test-eip"
	eip.Spec.Address = "1.1.1.0/24"
	if t.Layer2 {
		eip.Spec.Protocol = constant.PorterProtocolLayer2
	}
	Expect(t.K8sClient.Create(context.TODO(), eip)).NotTo(HaveOccurred())
	defer func() {
		t.K8sClient.Delete(context.TODO(), eip)
		WaitForDeletion(t.K8sClient, eip, 5*time.Second, 1*time.Minute)
	}()

	//create deployment
	nginxDeployment, err := t.prepareDeployment()
	Expect(err).ShouldNot(HaveOccurred())
	defer func() {
		t.K8sClient.Delete(context.TODO(), nginxDeployment)
		WaitForDeletion(t.K8sClient, nginxDeployment, 5*time.Second, 1*time.Minute)
	}()

	//apply service
	serviceStr := `{
		"kind": "Service",
		"apiVersion": "v1",
		"metadata": {
			"name": "test-app",
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
	service.Namespace = t.Namespace
	serviceType := types.NamespacedName{
		Namespace: t.Namespace,
		Name:      service.Name,
	}
	if t.Layer2 {
		service.Annotations[constant.PorterProtocolAnnotationKey] = constant.PorterProtocolLayer2
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
			log.Printf("Service EIP %v", service.Status.LoadBalancer.Ingress[0].IP)
			if ipnet.Contains(net.ParseIP(service.Status.LoadBalancer.Ingress[0].IP)) {
				return nil
			}
		}
		return fmt.Errorf("Failed to get correct ingress")
	}, 2*time.Minute, time.Second).Should(Succeed())

	if t.Layer2 {
	} else {
		//check route in bird
		Eventually(func() error {
			ips, err := GetServiceNodesIP(t.K8sClient, serviceType)
			if err != nil {
				return err
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
	}

	//inject test
	if t.InjectTest != nil {
		log.Printf("InjectTest begin")
		t.InjectTest()
		log.Printf("InjectTest end")
	}

	//CheckLog
	log, err := t.GetRouterLog()
	Expect(err).ShouldNot(HaveOccurred(), "Failed to get log of router")
	Expect(log).ShouldNot(ContainSubstring("error"))
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
