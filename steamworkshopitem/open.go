package steamworkshopitem

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/cache"
	xtar "github.com/frantjc/x/archive/tar"
)

type Opts struct {
	username, password string
	platformType       steamcmd.PlatformType
}

type Opt func(*Opts)

func WithURLValues(query url.Values) Opt {
	return func(o *Opts) {
		for _, opt := range []Opt{
			WithAccount(
				query.Get("username"),
				query.Get("password"),
			),
			WithPlatformType(
				steamcmd.PlatformType(query.Get("platformtype")),
			),
		} {
			opt(o)
		}
	}
}

func URLValues(o *Opts) url.Values {
	query := url.Values{}
	query.Add("username", o.username)
	query.Add("password", o.password)
	query.Add("platformtype", o.platformType.String())
	return query
}

func WithPlatformType(platformType steamcmd.PlatformType) Opt {
	return func(o *Opts) {
		o.platformType = platformType
	}
}

func WithAccount(username, password string) Opt {
	return func(o *Opts) {
		o.username = username
		o.password = password
	}
}

const (
	Scheme = "steamworkshopitem"
)

func Open(ctx context.Context, appID, publishedFileID int, opts ...Opt) (io.ReadCloser, error) {
	o := &Opts{
		platformType: steamcmd.DefaultPlatformType,
	}

	for _, opt := range opts {
		opt(o)
	}

	prompt, err := steamcmd.Start(ctx)
	if err != nil {
		return nil, err
	}
	defer prompt.Close(ctx)

	installDir := filepath.Join(cache.Dir, Scheme, o.platformType.String(), fmt.Sprint(appID), fmt.Sprint(publishedFileID))

	if err := prompt.ForceInstallDir(ctx, installDir); err != nil {
		return nil, err
	}

	if err := prompt.Login(ctx, steamcmd.WithAccount(o.username, o.password)); err != nil {
		return nil, err
	}

	if o.platformType != "" {
		if err := prompt.ForcePlatformType(ctx, o.platformType); err != nil {
			return nil, err
		}
	}

	if err := prompt.WorkshopDownloadItem(ctx, appID, publishedFileID); err != nil {
		return nil, err
	}

	return xtar.Compress(
		filepath.Join(
			installDir,
			"steamapps/workshop/content",
			fmt.Sprint(appID),
			fmt.Sprint(publishedFileID),
		),
	), nil
}
