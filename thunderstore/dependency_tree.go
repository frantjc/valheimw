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
	seenPkgs map[string]struct{}
}

func (b *depTreeBldr) buildDependencyTree(ctx context.Context, pkgNames ...string) ([]Package, error) {
	pkgs := []Package{}

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
			b.seenPkgs = map[string]struct{}{}
		}

		if _, found := b.seenPkgs[pkg.String()]; found {
			continue
		}

		if b.client == nil {
			b.client = DefaultClient
		}

		pkg, err = b.client.GetPackage(ctx, pkg)
		if err != nil {
			return nil, err
		}

		if _, found := b.seenPkgs[pkg.String()]; found {
			continue
		}

		b.seenPkgs[pkg.String()] = struct{}{}

		pkgs = append(pkgs, *pkg)
		deps := pkg.Dependencies
		if pkg.Latest != nil {
			deps = pkg.Latest.Dependencies
		}

		depPkgs, err := b.buildDependencyTree(ctx, deps...)
		if err != nil {
			return nil, err
		}

		pkgs = append(pkgs, depPkgs...)
	}

	return dedupePkgs(pkgs), nil
}

func dedupePkgs(pkgs []Package) []Package {
	seenPkgs := map[string]Package{}

	for _, pkg := range pkgs {
		key := fmt.Sprintf("%s|%s", pkg.Namespace, pkg.Name)
		if seenPkg, ok := seenPkgs[key]; !ok || seenPkg.VersionNumber == "" {
			seenPkgs[key] = pkg
		}
	}

	var (
		deduped = make([]Package, len(seenPkgs))
		i       = 0
	)
	for _, pkg := range seenPkgs {
		deduped[i] = pkg
		i++
	}

	return deduped
}
