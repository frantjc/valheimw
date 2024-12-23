package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/frantjc/sindri/ctr"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
)

type Client struct {
	*client.Client
}

var _ ctr.ImageClient = &Client{}

func (c *Client) Pull(ctx context.Context, ref string) (ctr.Image, error) {
	pref, err := name.ParseReference(ref)
	if err != nil {
		return nil, err
	}

	rc, err := c.Client.ImagePull(ctx, pref.String(), image.PullOptions{})
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	if _, err = io.Copy(io.Discard, rc); err != nil {
		return nil, err
	}

	return daemon.Image(pref, daemon.WithContext(ctx), daemon.WithClient(c))
}
