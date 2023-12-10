package xurl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func Open(ctx context.Context, file string) (io.ReadCloser, error) {
	u, err := url.Parse(file)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "", "file":
		return os.Open(u.Path)
	case "http", "https":
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return nil, err
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		return res.Body, nil
	}

	return nil, fmt.Errorf("read unknown scheme: %s", u.Scheme)
}
