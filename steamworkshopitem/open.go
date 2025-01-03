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
	login        steamcmd.Login
	platformType steamcmd.PlatformType
}

type Opt func(*Opts)

func WithURLValues(query url.Values) Opt {
	return func(o *Opts) {
		for _, opt := range []Opt{
			WithLogin(
				query.Get("username"),
				query.Get("password"),
				query.Get("steamguardcode"),
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
	query.Add("username", o.login.Username)
	query.Add("password", o.login.Password)
	query.Add("steamguardcode", o.login.SteamGuardCode)
	query.Add("platformtype", o.platformType.String())
	return query
}

func WithPlatformType(platformType steamcmd.PlatformType) Opt {
	return func(o *Opts) {
		o.platformType = platformType
	}
}

func WithLogin(username, password, steamGuardCode string) Opt {
	return func(o *Opts) {
		o.login = steamcmd.Login{
			Username:       username,
			Password:       password,
			SteamGuardCode: steamGuardCode,
		}
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
	installDir := filepath.Join(cache.Dir, Scheme, o.platformType.String(), fmt.Sprint(appID), fmt.Sprint(publishedFileID))

	commands := []steamcmd.Command{
		steamcmd.ForceInstallDir(installDir),
		o.login,
	}
	if o.platformType != "" {
		commands = append(commands, steamcmd.ForcePlatformType(o.platformType))
	}
	commands = append(commands,
		&steamcmd.WorkshopDownloadItem{
			AppID:           appID,
			PublishedFileID: publishedFileID,
		},
	)

	if err := steamcmd.Run(ctx, commands...); err != nil {
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
