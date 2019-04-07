package e2eutil

import (
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

const (
	BGPImageName = "magicsong/gobgp"
)

func RunGoBGPContainer(configPath string) (string, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", err
	}
	hostConfig := &container.HostConfig{
		NetworkMode: "host",
		Binds:       []string{"/root/bgp/test.toml:/etc/gobgp/gobgp.conf"},
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: BGPImageName,
	}, hostConfig, nil, "")
	if err != nil {
		return "", err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func GetContainerLog(containerID string) (string, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", err
	}

	options := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	// Replace this ID with a container that really exists
	out, err := cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", err
	}
	l, err := ioutil.ReadAll(out)
	if err != nil {
		log.Println("Error in read log")
		return "", err
	}
	return string(l), nil
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
		log.Println("Failed to stop container,err: ", err.Error())
		return err
	}
	return cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
}

func CheckBGPRoute() (string, error) {
	cmd := exec.Command("gobgp", "global", "rib")
	bytes, err := cmd.CombinedOutput()
	return string(bytes), err
}
