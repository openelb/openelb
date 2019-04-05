package e2eutil

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kubesphere/test-infra/bazel-test-infra/external/go_sdk/src/html/template"
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

func (t *TestCase) StartRemoteRoute(id chan<- string, errCh chan<- error) {
	//route config
	routeGeneratedConfig := "/tmp/route.toml"
	err := WriteConfig(t.RouterTemplatePath, routeGeneratedConfig, t)
	if err != nil {
		errCh <- err
		return
	}
	err = ScpFileToRemote(routeGeneratedConfig, t.RouterConfigPath, t.RouterIP)
	if err != nil {
		log.Printf("Error in transfer router config, error: %s", err.Error())
		errCh <- err
		return
	}
	//start a container this will block until container end
	RunGoBGPContainer(t.RouterConfigPath, id, errCh)
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
	return nil
}

func (t *TestCase) GetRouterLog() (string, error) {
	if t.routeContainerID == "" {
		return "", routerContainerNotExist
	}
	return GetContainerLog(t.routeContainerID)
}
