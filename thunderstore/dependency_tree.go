package thunderstore

import (
	"context"
	"fmt"
	"net/url"
)

func DependencyTree(ctx context.Context, pkgNames ...string) ([]Package, error) {
	return new(depTreeBldr).buildDependencyTree(ctx, pkgNames...)
}

type depTreeBldr struct {
	client   *Client
	seenPkgs map[string]Package
}

func (b *depTreeBldr) buildDependencyTree(ctx context.Context, pkgNames ...string) ([]Package, error) {
	for _, pkgName := range pkgNames {
		u, err := url.Parse(pkgName)
		if err != nil {
			return nil, err
		}

		if u.Scheme != "" && u.Scheme != Scheme {
			return nil, fmt.Errorf("unsupported scheme %s", u.Scheme)
		}

		pkg, err := ParsePackage(fmt.Sprintf("%s%s", u.Host, u.Path))
		if err != nil {
			return nil, err
		}

		if b == nil {
			b = new(depTreeBldr)
		}

		if b.seenPkgs == nil {
			b.seenPkgs = map[string]Package{}
		}

		if _, found := b.seenPkgs[pkg.Versionless()]; found && pkg.VersionNumber == "" {
			continue
		}

		if b.client == nil {
			b.client = DefaultClient
		}

		p, err := b.client.GetPackage(ctx, pkg)
		if err != nil {
			return nil, err
		}

		b.seenPkgs[pkg.Versionless()] = *p

		deps := pkg.Dependencies
		if pkg.Latest != nil {
			deps = pkg.Latest.Dependencies
		}

		if _, err = b.buildDependencyTree(ctx, deps...); err != nil {
			return nil, err
		}
	}

	var (
		pkgs = make([]Package, len(b.seenPkgs))
		i    int
	)
	for _, seenPkg := range b.seenPkgs {
		pkgs[i] = seenPkg
		i++
	}

	return pkgs, nil
}
