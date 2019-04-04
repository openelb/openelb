package e2eutil

import (
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
}

func (t *TestCase) WaitForControllerUp() error {
	return WaitForController(t.K8sClient, t.Namespace, managerName, 5*time.Second, 2*time.Minute)
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

func (t *TestCase) StartRemoteRoute() (string, error) {
	//route config
	routeGeneratedConfig := "/tmp/route.toml"
	err := WriteConfig(t.RouterTemplatePath, routeGeneratedConfig, t)
	if err != nil {
		return "", err
	}
	err = ScpFileToRemote(routeGeneratedConfig, t.RouterConfigPath, t.RouterIP)
	if err != nil {
		log.Printf("Error in transfer router config, error: %s", err.Error())
		return "", err
	}
	//start a container
	containerid, err := RunGoBGPContainer(t.RouterConfigPath)
	if err != nil {
		log.Println("Failed to start remote router")
		return "", err
	}
	return containerid, nil
}

func (t *TestCase) DeployYaml(userConfigPath, yamlPath string) error {
	//generate config
	err := WriteConfig(t.ControllerTemplatePath, userConfigPath, t)
	if err != nil {
		log.Println("Failed to generate controller config")
		return err
	}
	//kustomize
	err = KustomizeBuild(t.KustomizePath, yamlPath)
	if err != nil {
		log.Println("Kustomize failed")
		return err
	}
	return KubectlApply(yamlPath)
}
