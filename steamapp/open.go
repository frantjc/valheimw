package steamapp

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/valheimw/internal/appinfoutil"
	"github.com/frantjc/valheimw/internal/cache"
	xtar "github.com/frantjc/x/archive/tar"
)

const (
	DefaultBranchName = "public"
)

type OpenOpts struct {
	Beta, BetaPassword string
	Login              steamcmd.Login
	PlatformType       steamcmd.PlatformType
	LaunchType         string
}

func (o *OpenOpts) Apply(opts *OpenOpts) {
	if o.Beta != "" {
		opts.Beta = o.Beta
	}
	if o.BetaPassword != "" {
		opts.BetaPassword = o.BetaPassword
	}
	if o.Login.Username != "" {
		opts.Login = o.Login
	}
	if o.PlatformType != "" {
		opts.PlatformType = o.PlatformType
	}
	opts.LaunchType = o.LaunchType
}

func (o *OpenOpts) getInstallDir(appID int) string {
	branchName := DefaultBranchName
	if o.Beta != "" {
		branchName = o.Beta
	}
	return filepath.Join(cache.Dir, Scheme, o.PlatformType.String(), fmt.Sprint(appID), branchName)
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
		Beta:         query.Get("beta"),
		BetaPassword: query.Get("betapassword"),
		PlatformType: steamcmd.PlatformType(query.Get("platformtype")),
		LaunchType:   query.Get("launchtype"),
	}
}

func URLValues(o *OpenOpts) url.Values {
	query := url.Values{}
	query.Add("username", o.Login.Username)
	query.Add("password", o.Login.Password)
	query.Add("steamguardcode", o.Login.SteamGuardCode)
	query.Add("beta", o.Beta)
	query.Add("betapassword", o.BetaPassword)
	query.Add("platformtype", o.PlatformType.String())
	query.Add("launchtypes", o.LaunchType)
	return query
}

const (
	Scheme = "steamapp"
)

func Open(ctx context.Context, appID int, opts ...OpenOpt) (io.ReadCloser, error) {
	o := &OpenOpts{
		PlatformType: steamcmd.DefaultPlatformType,
	}

	for _, opt := range opts {
		opt.Apply(o)
	}

	appInfo, err := appinfoutil.GetAppInfo(ctx, appID,
		appinfoutil.WithLogin(o.Login.Username, o.Login.Password, o.Login.SteamGuardCode),
	)
	if err != nil {
		return nil, err
	}

	branchName := DefaultBranchName
	if o.Beta != "" {
		branchName = o.Beta
	}

	branch, ok := appInfo.Depots.Branches[branchName]
	if !ok {
		return nil, fmt.Errorf("branch %s not found", branchName)
	}

	if branch.PwdRequired && o.BetaPassword == "" {
		return nil, fmt.Errorf("steamapp %d branch %s requires a password", appID, branchName)
	}

	installDir := o.getInstallDir(appID)

	commands := []steamcmd.Command{
		steamcmd.ForceInstallDir(installDir),
		o.Login,
	}
	if o.PlatformType != "" {
		commands = append(commands, steamcmd.ForcePlatformType(o.PlatformType))
	}
	commands = append(commands, steamcmd.AppUpdate{
		AppID:        appID,
		Beta:         o.Beta,
		BetaPassword: o.BetaPassword,
	})

	if err := steamcmd.Run(ctx, commands...); err != nil {
		return nil, err
	}

	return xtar.Compress(installDir), nil
}
