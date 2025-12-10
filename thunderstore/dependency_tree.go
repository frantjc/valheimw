package thunderstore

import (
	"context"
	"fmt"
	"net/url"
)

func DependencyTree(ctx context.Context, pkgNames ...string) ([]Package, error) {
	b := new(depTreeBldr)

	if err := b.buildDependencyTree(ctx, nil, pkgNames...); err != nil {
		return nil, err
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

type depTreeBldr struct {
	client   *Client
	seenPkgs map[string]Package
}

func (b *depTreeBldr) buildDependencyTree(ctx context.Context, communityListings []CommunityListing, pkgNames ...string) error {
	for _, pkgName := range pkgNames {
		u, err := url.Parse(pkgName)
		if err != nil {
			return err
		}

		if u.Scheme != "" && u.Scheme != Scheme {
			return fmt.Errorf("unsupported scheme %s", u.Scheme)
		}

		pkg, err := ParsePackage(fmt.Sprintf("%s%s", u.Host, u.Path))
		if err != nil {
			return err
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
			return err
		}

		if p.CommunityListings == nil {
			p.CommunityListings = []CommunityListing{}
		}

		if communityListings != nil {
			p.CommunityListings = append(p.CommunityListings, communityListings...)
		}

		b.seenPkgs[p.Versionless()] = *p

		deps := p.Dependencies
		if p.Latest != nil {
			deps = p.Latest.Dependencies
		}

		if err = b.buildDependencyTree(ctx, p.CommunityListings, deps...); err != nil {
			return err
		}
	}

	return nil
}
