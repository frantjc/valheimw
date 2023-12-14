package xcontainerregistry

import (
	"io"

	xtar "github.com/frantjc/sindri/x/tar"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func LayerFromDir(dir string) (v1.Layer, error) {
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return xtar.Compress(dir), nil
	})
}
