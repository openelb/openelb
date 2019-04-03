package e2eutil

import (
	"text/template"

	"github.com/docker/docker/client"
)

type TestCase struct {
	ControllerAS   int
	ControllerIP   string
	ControllerPort int

	RouterIP       string
	RouterAS       int
	UsePortforward bool

	Client client.Client
}

func (t *TestCase) Start(routerConfigPath, controllerConfigPath string) error {
	t, err := template.New("routerConfig").ParseFiles(routerConfig)
	if err != nil {
		return err
	}
	//start a container

}
