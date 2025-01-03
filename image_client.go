package sindri

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/frantjc/sindri/ctr"
	"github.com/frantjc/sindri/ctr/docker"
	"github.com/frantjc/sindri/ctr/native"
)

func NewImageClient(ctx context.Context) ctr.ImageClient {
	if cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err == nil && cli != nil {
		if _, err := cli.Ping(context.Background()); err == nil {
			return &docker.Client{Client: cli}
		}
	}

	return &native.Client{}
}
