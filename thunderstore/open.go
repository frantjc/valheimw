package thunderstore

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/frantjc/valheimw/internal/cache"
	xtar "github.com/frantjc/x/archive/tar"
	xzip "github.com/frantjc/x/archive/zip"
	xio "github.com/frantjc/x/io"
	xslices "github.com/frantjc/x/slices"
)

type OpenOpts struct {
	Client *Client
}

type Opt interface {
	Apply(*OpenOpts)
}

func (o *OpenOpts) Apply(opts *OpenOpts) {
	if o.Client != nil {
		opts.Client = o.Client
	}
}

const (
	Scheme = "thunderstore"
)

func Open(ctx context.Context, pkg *Package, opts ...Opt) (io.ReadCloser, error) {
	o := &OpenOpts{
		Client: DefaultClient,
	}

	for _, opt := range opts {
		opt.Apply(o)
	}

	pkgZip, err := o.Client.GetPackageZip(ctx, pkg)
	if err != nil {
		return nil, err
	}
	defer pkgZip.Close()

	pkgZipRdr, err := zip.NewReader(pkgZip, pkgZip.Size())
	if err != nil {
		return nil, err
	}

	pkgZipRdr.File = xslices.Map(pkgZipRdr.File, func(f *zip.File, _ int) *zip.File {
		f.Name = strings.ReplaceAll(f.Name, "\\", "/")
		f.Name = strings.TrimPrefix(f.Name, pkg.Name)
		return f
	})

	var (
		baseDir    = filepath.Join(cache.Dir, Scheme, pkg.Namespace)
		installDir = filepath.Join(baseDir, pkg.Name, pkg.VersionNumber)
	)

	if err := xzip.Extract(pkgZipRdr, installDir); err != nil {
		return nil, err
	}

	rc := xtar.Compress(installDir)

	return xio.ReadCloser{
		Reader: rc,
		Closer: xio.CloserFunc(func() error {
			return errors.Join(rc.Close(), os.RemoveAll(baseDir))
		}),
	}, nil
}
