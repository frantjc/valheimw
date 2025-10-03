package steamapp

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"

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

	appID, err := strconv.Atoi(u.Host)
	if err != nil {
		return nil, err
	}

	return Open(
		ctx,
		appID,
		WithURLValues(u.Query()),
	)
}
