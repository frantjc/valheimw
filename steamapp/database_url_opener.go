package steamapp

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/frantjc/go-steamcmd"
)

type GettableBuildImageOpts struct {
	BaseImageRef string

	AptPkgs []string

	LaunchType   string
	PlatformType steamcmd.PlatformType

	Execs []string

	Entrypoint []string
	Cmd        []string
}

func (o *GettableBuildImageOpts) Apply(opts *BuildImageOpts) {
	if o == nil {
		return
	}
	if o.BaseImageRef != "" {
		opts.BaseImageRef = o.BaseImageRef
	}
	if len(o.AptPkgs) > 0 {
		opts.AptPkgs = o.AptPkgs
	}
	if o.LaunchType != "" {
		opts.LaunchType = o.LaunchType
	}
	if o.PlatformType.String() != "" {
		opts.PlatformType = o.PlatformType
	}
	if len(o.Execs) > 0 {
		opts.Execs = o.Execs
	}
	if len(o.Entrypoint) > 0 {
		opts.Entrypoint = o.Entrypoint
	}
	if len(o.Cmd) > 0 {
		opts.Cmd = o.Cmd
	}
}

type Database interface {
	GetBuildImageOpts(context.Context, int, string) (*GettableBuildImageOpts, error)
}

type DatabaseURLOpener interface {
	OpenDatabase(context.Context, *url.URL) (Database, error)
}

var (
	urlMux = map[string]DatabaseURLOpener{}
)

func RegisterDatabase(o DatabaseURLOpener, scheme string, schemes ...string) {
	for _, s := range append(schemes, scheme) {
		if _, ok := urlMux[s]; ok {
			panic("attempt to reregister scheme: " + s)
		}

		urlMux[s] = o
	}
}

func OpenDatabase(ctx context.Context, s string) (Database, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	o, ok := urlMux[strings.ToLower(u.Scheme)]
	if !ok {
		return nil, fmt.Errorf("no opener registered for scheme %s", u.Scheme)
	}

	return o.OpenDatabase(ctx, u)
}
