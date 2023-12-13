package util

import (
	"context"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// GetTDEngineHostName Get TDEngien docker host name
func GetTDEngineHostName() (string, string) {
	ctx := context.Background()

	// get client object
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	// get container list. this function is similar to docker ps
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})

	// include all containers that in runtime or showdown. ContainerList is similar to docker ps -a
	//containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		panic(err)
	}

	taosdockerhost := ""
	taosdockerport := ""
	// get all result
	for _, container := range containers {
		// fmt.Println(container.ID, container.Names, container.Created, container.Status, container.Ports)
		if strings.Contains(container.Names[0][1:], "taos") {
			// filter docker name . using TDEngine
			taosdockerhost = container.Names[0][1:]

			for _, v := range container.Ports {
				if v.PublicPort != 0 {
					taosdockerport = strconv.Itoa(int(v.PublicPort))
				}
			}
		}
	}

	return taosdockerhost, taosdockerport
}
