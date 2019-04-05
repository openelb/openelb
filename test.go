package main

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func main() {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	imageName := "magicsong/gobgp"
	hostConfig := &container.HostConfig{
		NetworkMode: "host",
		Binds:       []string{"/root/bgp/test.toml:/etc/gobgp/gobgp.conf"},
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
	}, hostConfig, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	fmt.Println(resp.ID)
}
