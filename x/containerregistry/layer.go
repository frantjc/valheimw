package xcontainerregistry

import (
	"io"

	xtar "github.com/frantjc/sindri/x/tar"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func LayerFromDir(dir string) (v1.Layer, error) {
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return xtar.Compress(dir), nil
	})
}

func AppendDirLayer(img v1.Image, dir string) (v1.Image, error) {
	layer, err := LayerFromDir(dir)
	if err != nil {
		return nil, err
	}

	return mutate.AppendLayers(img, layer)
}
