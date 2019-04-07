package e2eutil

import (
	"fmt"
	"log"
	"os"
	"text/template"
	"time"

	"github.com/kubesphere/porter/pkg/util"
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
func (t *TestCase) DeployYaml(userConfigPath string) error {
	//generate config
	err := WriteConfig(t.ControllerTemplatePath, userConfigPath, t)
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
