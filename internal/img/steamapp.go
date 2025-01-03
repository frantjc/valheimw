package img

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	reproduciblebuilds "github.com/frantjc/go-reproducible-builds"
	"github.com/frantjc/sindri"
	"github.com/frantjc/sindri/steamapp"
	xtar "github.com/frantjc/x/archive/tar"
	xio "github.com/frantjc/x/io"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

type BuildSteamappOpts struct {
	steamappOpts []steamapp.Opt
	baseImage    v1.Image
	baseImageRef string
	uname, gname string
}

type BuildSteamappOpt func(*BuildSteamappOpts)

func WithSteamappOpts(opts ...steamapp.Opt) BuildSteamappOpt {
	return func(o *BuildSteamappOpts) {
		o.steamappOpts = append(o.steamappOpts, opts...)
	}
}

func WithBaseImage(baseImage v1.Image) BuildSteamappOpt {
	return func(o *BuildSteamappOpts) {
		o.baseImageRef = ""
		o.baseImage = baseImage
	}
}

func WithBaseImageRef(baseImageRef string) BuildSteamappOpt {
	return func(o *BuildSteamappOpts) {
		o.baseImageRef = baseImageRef
	}
}

func WithUser(uname, gname string) BuildSteamappOpt {
	return func(o *BuildSteamappOpts) {
		o.uname = uname
		o.gname = gname
	}
}

func SteamappImage(ctx context.Context, appID int, opts ...BuildSteamappOpt) (v1.Image, error) {
	o := &BuildSteamappOpts{
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

	cfg, err := steamapp.ImageConfig(ctx, appID, &cfgf.Config, o.steamappOpts...)
	if err != nil {
		return nil, err
	}

	image, err := mutate.Config(baseImage, *cfg)
	if err != nil {
		return nil, err
	}

	layer, err := ReproducibleBuildLayerInDirFromOpener(
		func() (io.ReadCloser, error) {
			return steamapp.Open(
				ctx,
				appID,
				o.steamappOpts...,
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

func ReproducibleBuildLayerInDirFromOpener(o tarball.Opener, dir, uname, gname string) (v1.Layer, error) {
	if filepath.IsAbs(dir) {
		return nil, fmt.Errorf("dir must be a relative path: %s", dir)
	}

	return tarball.LayerFromOpener(
		func() (io.ReadCloser, error) {
			rc1, err := o()
			if err != nil {
				return nil, err
			}

			rc2 := xtar.ModifyHeaders(
				tar.NewReader(rc1),
				func(h *tar.Header) {
					//nolint:gosec
					h.Name = filepath.Join(dir, h.Name)
					h.ModTime = reproduciblebuilds.SourceDateEpoch
					h.Uname = uname
					h.Gname = gname
				},
			)

			return &xio.ReadCloser{
				Reader: rc2,
				Closer: xio.CloserFunc(func() error {
					return errors.Join(rc2.Close(), rc1.Close())
				}),
			}, nil
		},
		tarball.WithCompressedCaching,
	)
}
