package ctr

import (
	"context"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type Image = v1.Image

type ImageClient interface {
	Pull(context.Context, string) (Image, error)
}
