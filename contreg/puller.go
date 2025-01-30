package contreg

import (
	"context"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type (
	Manifest = v1.Manifest
	Blob     = v1.Layer
)

type Puller interface {
	HeadManifest(ctx context.Context, name string, reference string) error
	GetManifest(ctx context.Context, name string, reference string) (*Manifest, error)
	HeadBlob(ctx context.Context, name string, digest string) error
	GetBlob(ctx context.Context, name string, digest string) (Blob, error)
}
