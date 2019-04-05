package e2eutil

import (
	"io/ioutil"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

const (
	BGPImageName = "magicsong/gobgp"
)

func RunGoBGPContainer(configPath string, containerID chan<- string, errCh chan<- error) {
	ctx := context.TODO()
	cli, err := client.NewEnvClient()
	if err != nil {
		errCh <- err
		return
	}
	hostConfig := &container.HostConfig{
		NetworkMode: "host",
		Binds:       []string{configPath + ":/etc/gobgp/gobgp.conf"},
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: BGPImageName,
	}, hostConfig, nil, "")
	if err != nil {
		log.Println("Error in create container")
		errCh <- err
		return
	}
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Println("Error in start container")
		errCh <- err
		return
	}
	containerID <- resp.ID
	cli.ContainerWait(ctx, resp.ID)
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
		return err
	}
	return nil
	//return cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
}
