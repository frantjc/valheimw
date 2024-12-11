package sindri

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	xtar "github.com/frantjc/x/archive/tar"
)

type CacheableURLOpener interface {
	Open(context.Context, *url.URL) (io.ReadCloser, error)
	CacheKey(context.Context, *url.URL) (string, error)
}

var (
	urlMux = map[string]CacheableURLOpener{}
)

func Register(o CacheableURLOpener, scheme string, schemes ...string) {
	for _, s := range append(schemes, scheme) {
		if _, ok := urlMux[s]; ok {
			panic("attempt to reregister scheme: " + s)
		}

		urlMux[s] = o
	}
}

func CacheKey(ctx context.Context, s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}

	o, ok := urlMux[strings.ToLower(u.Scheme)]
	if !ok {
		return "", fmt.Errorf("no opener registered for scheme %s", u.Scheme)
	}

	return o.CacheKey(ctx, u)
}

func Open(ctx context.Context, s string) (io.ReadCloser, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	o, ok := urlMux[strings.ToLower(u.Scheme)]
	if !ok {
		return nil, fmt.Errorf("no opener registered for scheme %s", u.Scheme)
	}

	return o.Open(ctx, u)
}

func Extract(ctx context.Context, s, dir string) error {
	rc, err := Open(ctx, s)
	if err != nil {
		return err
	}
	defer rc.Close()

	return xtar.Extract(tar.NewReader(rc), dir)
}
