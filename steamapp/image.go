package steamapp

import (
	"context"
	"io"

	"github.com/frantjc/sindri"
	"github.com/frantjc/sindri/internal/imgutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
)

type ImageOpts struct {
	opts         []Opt
	baseImage    v1.Image
	baseImageRef string
	uname, gname string
}

type ImageOpt func(*ImageOpts)

func WithOpts(opts ...Opt) ImageOpt {
	return func(o *ImageOpts) {
		o.opts = append(o.opts, opts...)
	}
}

func WithBaseImage(baseImage v1.Image) ImageOpt {
	return func(o *ImageOpts) {
		o.baseImageRef = ""
		o.baseImage = baseImage
	}
}

func WithBaseImageRef(baseImageRef string) ImageOpt {
	return func(o *ImageOpts) {
		o.baseImageRef = baseImageRef
	}
}

func WithUser(uname, gname string) ImageOpt {
	return func(o *ImageOpts) {
		o.uname = uname
		o.gname = gname
	}
}

func NewImage(ctx context.Context, appID int, opts ...ImageOpt) (v1.Image, error) {
	o := &ImageOpts{
		baseImage: empty.Image,
	}

	for _, opt := range opts {
		opt(o)
	}

	baseImage := o.baseImage
	if o.baseImageRef != "" {
		var err error
		baseImage, err = sindri.NewImageClient().Pull(ctx, o.baseImageRef)
		if err != nil {
			return nil, err
		}
	}

	cfgf, err := baseImage.ConfigFile()
	if err != nil {
		return nil, err
	}

	cfg, err := NewImageConfig(ctx, appID, &cfgf.Config, o.opts...)
	if err != nil {
		return nil, err
	}

	image, err := mutate.Config(baseImage, *cfg)
	if err != nil {
		return nil, err
	}

	layer, err := imgutil.ReproducibleBuildLayerInDirFromOpener(
		func() (io.ReadCloser, error) {
			return Open(
				ctx,
				appID,
				o.opts...,
			)
		},
		cfg.WorkingDir,
		o.uname, o.gname,
	)
	if err != nil {
		return nil, err
	}

	image, err = mutate.AppendLayers(image, layer)
	if err != nil {
		return nil, err
	}

	return image, nil
}
