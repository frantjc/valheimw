package thunderstore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
)

type ClientOpt func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOpt {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithDir(dir string) ClientOpt {
	return func(c *Client) {
		c.dir = dir
	}
}

func NewClient(u *url.URL, opts ...ClientOpt) *Client {
	c := &Client{u, http.DefaultClient, os.TempDir()}

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

func (c *Client) GetPackageMetadata(ctx context.Context, p *Package) (*PackageMetadata, error) {
	meta := &PackageMetadata{}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		c.thunderstoreURL.JoinPath("/api/experimental/package").JoinPath(packageElems(p)...).String()+"/",
		nil,
	)
	if err != nil {
		return nil, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return meta, json.NewDecoder(res.Body).Decode(meta)
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
	return os.Remove(z.f.Name())
}

func (c *Client) GetPackageZip(ctx context.Context, p *Package) (*ZipReadableCloser, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		c.thunderstoreURL.JoinPath("/package/download").JoinPath(packageElems(p)...).String()+"/",
		nil,
	)
	if err != nil {
		return nil, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	f, err := os.CreateTemp(c.dir, p.Fullname()+"-*.zip")
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(f, res.Body); err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return &ZipReadableCloser{f, fi.Size()}, nil
}

func packageElems(p *Package) []string {
	return []string{p.Namespace, p.Name, p.Version}
}
