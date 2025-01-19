package imgutil

import (
	"archive/tar"
	"errors"
	"io"
	"path/filepath"

	reproduciblebuilds "github.com/frantjc/go-reproducible-builds"
	xtar "github.com/frantjc/x/archive/tar"
	xio "github.com/frantjc/x/io"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func ReproducibleBuildLayerInDirFromOpener(o tarball.Opener, dir, uname, gname string) (v1.Layer, error) {
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
