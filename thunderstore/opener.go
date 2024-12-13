package thunderstore

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/frantjc/sindri/internal/cache"
	xtar "github.com/frantjc/x/archive/tar"
	xzip "github.com/frantjc/x/archive/zip"
	xio "github.com/frantjc/x/io"
	xslice "github.com/frantjc/x/slice"
)

type Opts struct {
	client *Client
}

type Opt func(*Opts)

func WithClient(cli *Client) Opt {
	return func(o *Opts) {
		o.client = cli
	}
}

const (
	Scheme = "thunderstore"
)

func Open(ctx context.Context, pkg *Package, opts ...Opt) (io.ReadCloser, error) {
	o := &Opts{
		client: DefaultClient,
	}

	for _, opt := range opts {
		opt(o)
	}

	pkgZip, err := o.client.GetPackageZip(ctx, pkg)
	if err != nil {
		return nil, err
	}
	defer pkgZip.Close()

	pkgZipRdr, err := zip.NewReader(pkgZip, pkgZip.Size())
	if err != nil {
		return nil, err
	}

	pkgZipRdr.File = xslice.Map(pkgZipRdr.File, func(f *zip.File, _ int) *zip.File {
		f.Name = strings.ReplaceAll(f.Name, "\\", "/")
		f.Name = strings.TrimPrefix(f.Name, pkg.Name)
		return f
	})

	installDir := filepath.Join(cache.Dir, Scheme, pkg.Namespace, pkg.Name, pkg.VersionNumber)

	if err := xzip.Extract(pkgZipRdr, installDir); err != nil {
		return nil, err
	}

	rc := xtar.Compress(installDir)

	return xio.ReadCloser{
		Reader: rc,
		Closer: xio.CloserFunc(func() error {
			return errors.Join(rc.Close(), os.RemoveAll(installDir))
		}),
	}, nil
}
