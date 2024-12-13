package contreg

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Client interface {
	Pull(context.Context, name.Reference) (v1.Image, error)
}

type DockerClient struct {
	*client.Client
}

var _ Client = &DockerClient{}

func (c *DockerClient) Pull(ctx context.Context, ref name.Reference) (v1.Image, error) {
	rc, err := c.Client.ImagePull(ctx, ref.String(), image.PullOptions{})
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	if _, err = io.Copy(io.Discard, rc); err != nil {
		return nil, err
	}

	return daemon.Image(ref, daemon.WithContext(ctx), daemon.WithClient(c))
}

type RemoteClient struct{}

var _ Client = &RemoteClient{}

func (*RemoteClient) Pull(ctx context.Context, ref name.Reference) (v1.Image, error) {
	return remote.Image(ref, remote.WithContext(ctx))
}

var (
	DefaultClient = func() Client {
		if cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err == nil && cli != nil {
			if _, err := cli.Ping(context.Background()); err == nil {
				return &DockerClient{cli}
			}
		}

		return &RemoteClient{}
	}()
)
