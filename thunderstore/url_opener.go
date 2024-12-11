package thunderstore

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/frantjc/sindri"
)

func init() {
	sindri.Register(
		new(CacheableURLOpener),
		Scheme,
	)
}

type CacheableURLOpener struct{}

func (o *CacheableURLOpener) CacheKey(ctx context.Context, u *url.URL) (string, error) {
	if u.Scheme != Scheme {
		return "", fmt.Errorf("invalid scheme %s, expected %s", u.Scheme, Scheme)
	}

	pkg, err := ParsePackage(
		fmt.Sprintf("%s%s", u.Host, u.Path),
	)
	if err != nil {
		return "", err
	}

	return CacheKey(ctx, pkg)
}

func (o *CacheableURLOpener) Open(ctx context.Context, u *url.URL) (io.ReadCloser, error) {
	if u.Scheme != Scheme {
		return nil, fmt.Errorf("invalid scheme %s, expected %s", u.Scheme, Scheme)
	}

	pkg, err := ParsePackage(
		fmt.Sprintf("%s%s", u.Host, u.Path),
	)
	if err != nil {
		return nil, err
	}

	return Open(ctx, pkg)
}
