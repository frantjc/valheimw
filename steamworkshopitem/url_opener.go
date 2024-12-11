package steamworkshopitem

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"

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

	appID, err := strconv.Atoi(u.Host)
	if err != nil {
		return "", err
	}

	if u.Path == "" {
		return "", fmt.Errorf("empty URL path does not contain a published file ID")
	}

	publishedFileID, err := strconv.Atoi(u.Path[1:])
	if err != nil {
		return "", err
	}

	return CacheKey(
		ctx,
		appID,
		publishedFileID,
		WithURLValues(u.Query()),
	)
}

func (o *CacheableURLOpener) Open(ctx context.Context, u *url.URL) (io.ReadCloser, error) {
	if u.Scheme != Scheme {
		return nil, fmt.Errorf("invalid scheme %s, expected %s", u.Scheme, Scheme)
	}

	appID, err := strconv.Atoi(u.Host)
	if err != nil {
		return nil, err
	}

	if u.Path == "" {
		return nil, fmt.Errorf("empty URL path does not contain a published file ID")
	}

	publishedFileID, err := strconv.Atoi(u.Path[1:])
	if err != nil {
		return nil, err
	}

	return Open(
		ctx,
		appID,
		publishedFileID,
		WithURLValues(u.Query()),
	)
}
