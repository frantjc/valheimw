package podman

import (
	"context"
	"io"

	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/frantjc/sindri/ctr"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

type Client struct {
	context.Context
}

var _ ctr.ImageClient = &Client{}

func (c *Client) Pull(ctx context.Context, ref string) (ctr.Image, error) {
	refs, err := images.Pull(c.Context, ref, nil)
	if err != nil {
		return nil, err
	}

	tag, err := name.NewTag(ref)
	if err != nil {
		return nil, err
	}

	return tarball.Image(
		func() (io.ReadCloser, error) {
			pr, pw := io.Pipe()

			if err = images.Export(c.Context, refs, pw, &images.ExportOptions{
				Compress: &[]bool{true}[0],
			}); err != nil {
				return nil, err
			}

			return pr, nil
		},
		&tag,
	)
}
