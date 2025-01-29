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

type OpenOpts struct {
	Login        steamcmd.Login
	PlatformType steamcmd.PlatformType
}

func (o *OpenOpts) Apply(opts *OpenOpts) {
	if o.Login.Username != "" {
		opts.Login = o.Login
	}
	if o.PlatformType != "" {
		opts.PlatformType = o.PlatformType
	}
}

type OpenOpt interface {
	Apply(*OpenOpts)
}

func WithURLValues(query url.Values) OpenOpt {
	return &OpenOpts{
		Login: steamcmd.Login{
			Username:       query.Get("username"),
			Password:       query.Get("password"),
			SteamGuardCode: query.Get("steamguardcode"),
		},
		PlatformType: steamcmd.PlatformType(query.Get("platformtype")),
	}
}

func URLValues(o *OpenOpts) url.Values {
	query := url.Values{}
	query.Add("username", o.Login.Username)
	query.Add("password", o.Login.Password)
	query.Add("steamguardcode", o.Login.SteamGuardCode)
	query.Add("platformtype", o.PlatformType.String())
	return query
}

const (
	Scheme = "steamworkshopitem"
)

func Open(ctx context.Context, appID, publishedFileID int, opts ...OpenOpt) (io.ReadCloser, error) {
	o := &OpenOpts{
		PlatformType: steamcmd.DefaultPlatformType,
	}

	for _, opt := range opts {
		opt.Apply(o)
	}
	installDir := filepath.Join(cache.Dir, Scheme, o.PlatformType.String(), fmt.Sprint(appID), fmt.Sprint(publishedFileID))

	commands := []steamcmd.Command{
		steamcmd.ForceInstallDir(installDir),
		o.Login,
	}
	if o.PlatformType != "" {
		commands = append(commands, steamcmd.ForcePlatformType(o.PlatformType))
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
