package e2eutil

import (
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

const (
	BGPImageName = "magicsong/gobgp"
)

func RunGoBGPContainer(configPath string) (string, error) {
	cmd := exec.Command("docker", "run", "-d", "-v", configPath+":/etc/gobgp/gobgp.conf", "--net=host", "magicsong/bgp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func StopGoBGPContainer(containerID string) error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	timeout := time.Second * 20
	err = cli.ContainerStop(ctx, containerID, &timeout)
	if err != nil {
		return err
	}
	return nil
	//return cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
}
