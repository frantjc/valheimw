package native

import (
	"context"

	"github.com/frantjc/sindri/ctr"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Client struct{}

var _ ctr.ImageClient = &Client{}

func (*Client) Pull(ctx context.Context, ref string) (ctr.Image, error) {
	pref, err := name.ParseReference(ref)
	if err != nil {
		return nil, err
	}

	return remote.Image(pref, remote.WithContext(ctx))
}
