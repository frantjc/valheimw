package thunderstore

import (
	"context"
)

func CacheKey(ctx context.Context, pkg *Package, opts ...Opt) (string, error) {
	o := &Opts{
		client: DefaultClient,
	}

	for _, opt := range opts {
		opt(o)
	}

	pkg, err := o.client.GetPackage(ctx, pkg)
	if err != nil {
		return "", err
	}

	if pkg.Latest != nil {
		return pkg.Latest.FullName, nil
	}

	return pkg.FullName, nil
}
