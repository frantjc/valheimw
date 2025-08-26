package trivy

import (
	"context"
	"fmt"
	"net/url"

	"github.com/frantjc/sindri/internal/controller"
)

const Scheme = "trivy"

func init() {
	controller.RegisterImageScanner(
		new(URLOpener),
		Scheme,
	)
}

type URLOpener struct{}

func (o *URLOpener) Open(ctx context.Context, u *url.URL) (controller.ImageScanner, error) {
	if u.Scheme != Scheme {
		return nil, fmt.Errorf("invalid scheme %s, expected %s", u.Scheme, Scheme)
	}

	q := u.Query()

	path := u.Path
	if path == "/" {
		path = ""
	}

	return NewScanner(ctx, &ScannerOpts{
		DBRepositories: q["dbrepos"],
		CacheDir:       path,
	})
}
