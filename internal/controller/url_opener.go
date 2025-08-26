package controller

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type ImageScannerURLOpener interface {
	Open(context.Context, *url.URL) (ImageScanner, error)
}

var (
	urlMux = map[string]ImageScannerURLOpener{}
)

func RegisterImageScanner(o ImageScannerURLOpener, scheme string, schemes ...string) {
	for _, s := range append(schemes, scheme) {
		if _, ok := urlMux[s]; ok {
			panic("attempt to reregister scheme: " + s)
		}

		urlMux[s] = o
	}
}

func OpenImageScanner(ctx context.Context, s string) (ImageScanner, error) {
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
