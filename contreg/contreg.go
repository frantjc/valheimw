package contreg

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Reader interface {
	Read(context.Context, name.Reference) (v1.Image, error)
}

type DockerClient struct {
	*client.Client
}

var _ Reader = &DockerClient{}

func (c *DockerClient) Read(ctx context.Context, ref name.Reference) (v1.Image, error) {
	return daemon.Image(ref, daemon.WithContext(ctx), daemon.WithClient(c))
}

type RemoteClient struct{}

var _ Reader = &RemoteClient{}

func (*RemoteClient) Read(ctx context.Context, ref name.Reference) (v1.Image, error) {
	return remote.Image(ref, remote.WithContext(ctx))
}

var (
	DefaultClient = func() Reader {
		if cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err == nil && cli != nil {
			return &DockerClient{cli}
		}

		return &RemoteClient{}
	}()
)
