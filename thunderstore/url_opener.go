package thunderstore

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/frantjc/valheimw"
)

func init() {
	valheimw.Register(
		new(URLOpener),
		Scheme,
	)
}

type URLOpener struct{}

func (o *URLOpener) Open(ctx context.Context, u *url.URL) (io.ReadCloser, error) {
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
