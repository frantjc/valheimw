package steamapp

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

const (
	DefaultBranchName = "public"
)

type Opts struct {
	installDir         string
	beta, betaPassword string
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
			WithBeta(
				query.Get("beta"),
				query.Get("betapassword"),
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
	query.Add("beta", o.beta)
	query.Add("betapassword", o.betaPassword)
	query.Add("platformtype", o.platformType.String())
	return query
}

func WithInstallDir(dir string) Opt {
	return func(o *Opts) {
		o.installDir = dir
	}
}

func WithBeta(beta, betaPassword string) Opt {
	return func(o *Opts) {
		o.beta = beta
		o.betaPassword = betaPassword
	}
}

func WithPlatformType(platformType steamcmd.PlatformType) Opt {
	return func(o *Opts) {
		if platformType != "" {
			o.platformType = platformType
		}
	}
}

func WithAccount(username, password string) Opt {
	return func(o *Opts) {
		o.username = username
		o.password = password
	}
}

const (
	Scheme = "steamapp"
)

func Open(ctx context.Context, appID int, opts ...Opt) (io.ReadCloser, error) {
	o := &Opts{
		platformType: steamcmd.DefaultPlatformType,
	}

	for _, opt := range opts {
		opt(o)
	}

	branchName := DefaultBranchName
	if o.beta != "" {
		branchName = o.beta
	}

	prompt, err := steamcmd.Start(ctx)
	if err != nil {
		return nil, err
	}
	defer prompt.Close(ctx)

	installDir := filepath.Join(cache.Dir, Scheme, o.platformType.String(), fmt.Sprint(appID), branchName)

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

	appInfo, err := prompt.AppInfoPrint(ctx, appID)
	if err != nil {
		return nil, err
	}

	branch, ok := appInfo.Depots.Branches[branchName]
	if !ok {
		return nil, fmt.Errorf("branch %s not found", branchName)
	}

	if branch.PwdRequired && o.betaPassword == "" {
		return nil, fmt.Errorf("steamapp %d branch %s requires a password", appID, branchName)
	}

	if err := prompt.AppUpdate(ctx, appID, steamcmd.WithBeta(o.beta, o.betaPassword)); err != nil {
		return nil, err
	}

	return xtar.Compress(installDir), nil
}
