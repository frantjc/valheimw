package thunderstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/frantjc/sindri/internal/cache"
)

var (
	DefaultURL = func() *url.URL {
		u, err := url.Parse("https://thunderstore.io/")
		if err != nil {
			panic(err)
		}

		return u
	}()
	DefaultClient = NewClient()
)

type ClientOpt func(*Client)

func WithDir(dir string) ClientOpt {
	return func(c *Client) {
		c.dir = dir
	}
}

func WithHTTPClient(httpClient *http.Client) ClientOpt {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithURL(u *url.URL) ClientOpt {
	return func(c *Client) {
		c.thunderstoreURL = u
	}
}

func NewClient(opts ...ClientOpt) *Client {
	c := &Client{DefaultURL, http.DefaultClient, filepath.Join(cache.Dir, Scheme)}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type Client struct {
	thunderstoreURL *url.URL
	httpClient      *http.Client
	dir             string
}

func (c *Client) GetPackage(ctx context.Context, p *Package) (*Package, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		fmt.Sprintf("%s/", c.thunderstoreURL.JoinPath("/api/experimental/package", p.Namespace, p.Name, p.VersionNumber).String()),
		nil,
	)
	if err != nil {
		return nil, err
	}

	pkg := &Package{}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(pkg); err != nil {
		return nil, err
	}

	if pkg.Detail == "Not found." {
		return nil, fmt.Errorf("package %s not found", p)
	}

	return pkg, nil
}

type ZipReadableCloser struct {
	f    *os.File
	size int64
}

func (z *ZipReadableCloser) ReadAt(b []byte, off int64) (int, error) {
	return z.f.ReadAt(b, off)
}

func (z *ZipReadableCloser) Size() int64 {
	return z.size
}

func (z *ZipReadableCloser) Close() error {
	return z.f.Close()
}

func (z *ZipReadableCloser) Remove() error {
	return os.Remove(z.f.Name())
}

func (c *Client) GetPackageZip(ctx context.Context, p *Package) (*ZipReadableCloser, error) {
	zipFilePath := filepath.Join(c.dir, fmt.Sprintf("%s.zip", p))

	fi, err := os.Stat(zipFilePath)
	if err == nil {
		f, err := os.Open(zipFilePath)
		if err != nil {
			return nil, err
		}

		return &ZipReadableCloser{f, fi.Size()}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if err = os.MkdirAll(c.dir, 0750); err != nil {
		return nil, err
	}

	u := ""
	switch {
	case p.Latest != nil && p.Latest.DownloadURL != nil && p.Latest.DownloadURL.URL != nil:
		u = p.Latest.DownloadURL.String()
	case p.DownloadURL != nil && p.DownloadURL.URL != nil:
		u = p.DownloadURL.String()
	case p.VersionNumber != "":
		u = fmt.Sprintf("%s/", c.thunderstoreURL.JoinPath("/package/download", p.Namespace, p.Name, p.VersionNumber).String())
	default:
		pkg, err := c.GetPackage(ctx, p)
		if err != nil {
			return nil, err
		}
		pkg.FullName = pkg.Latest.FullName

		return c.GetPackageZip(ctx, pkg)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	f, err := os.Create(zipFilePath)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(f, res.Body); err != nil {
		return nil, err
	}

	if res.ContentLength >= 0 {
		return &ZipReadableCloser{f, res.ContentLength}, nil
	}

	fi, err = f.Stat()
	if err != nil {
		return nil, err
	}

	return &ZipReadableCloser{f, fi.Size()}, nil
}
