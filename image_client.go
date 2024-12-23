package sindri

import (
	"context"
	"os"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/docker/docker/client"
	"github.com/frantjc/sindri/ctr"
	"github.com/frantjc/sindri/ctr/docker"
	"github.com/frantjc/sindri/ctr/native"
	"github.com/frantjc/sindri/ctr/podman"
)

func NewImageClient(ctx context.Context) ctr.ImageClient {
	if cctx, err := bindings.NewConnection(context.Background(), os.Getenv("CONTAINER_HOST")); err == nil {
		return &podman.Client{Context: cctx}
	}

	if cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err == nil && cli != nil {
		if _, err := cli.Ping(context.Background()); err == nil {
			return &docker.Client{Client: cli}
		}
	}

	return &native.Client{}
}
