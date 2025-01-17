package steamapp

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/frantjc/go-kv"
	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/appinfoutil"
	"github.com/frantjc/sindri/internal/cache"
	xtar "github.com/frantjc/x/archive/tar"
)

const (
	DefaultBranchName = "public"
)

type Opts struct {
	installDir         string
	beta, betaPassword string
	login              steamcmd.Login
	platformType       steamcmd.PlatformType
	launchTypes        []string
	store              kv.Store
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
			WithBeta(
				query.Get("beta"),
				query.Get("betapassword"),
			),
			WithPlatformType(
				steamcmd.PlatformType(query.Get("platformtype")),
			),
			WithLaunchTypes(strings.Split(query.Get("launchtypes"), ",")...),
		} {
			opt(o)
		}
	}
}

func WithStore(store kv.Store) Opt {
	return func(o *Opts) {
		o.store = store
	}
}

func URLValues(o *Opts) url.Values {
	query := url.Values{}
	query.Add("username", o.login.Username)
	query.Add("password", o.login.Password)
	query.Add("steamguardcode", o.login.SteamGuardCode)
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

func WithLogin(username, password, steamGuardCode string) Opt {
	return func(o *Opts) {
		o.login = steamcmd.Login{
			Username:       username,
			Password:       password,
			SteamGuardCode: steamGuardCode,
		}
	}
}

func WithLaunchTypes(launchTypes ...string) Opt {
	return func(o *Opts) {
		o.launchTypes = append(o.launchTypes, launchTypes...)
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

	appInfo, err := appinfoutil.GetAppInfo(ctx, appID,
		appinfoutil.WithLogin(o.login.Username, o.login.Password, o.login.SteamGuardCode),
		appinfoutil.WithStore(o.store),
	)
	if err != nil {
		return nil, err
	}

	branchName := DefaultBranchName
	if o.beta != "" {
		branchName = o.beta
	}

	branch, ok := appInfo.Depots.Branches[branchName]
	if !ok {
		return nil, fmt.Errorf("branch %s not found", branchName)
	}

	if branch.PwdRequired && o.betaPassword == "" {
		return nil, fmt.Errorf("steamapp %d branch %s requires a password", appID, branchName)
	}

	installDir := filepath.Join(cache.Dir, Scheme, o.platformType.String(), fmt.Sprint(appID), branchName)

	commands := []steamcmd.Command{
		steamcmd.ForceInstallDir(installDir),
		o.login,
	}
	if o.platformType != "" {
		commands = append(commands, steamcmd.ForcePlatformType(o.platformType))
	}
	commands = append(commands, steamcmd.AppUpdate{
		AppID:        appID,
		Beta:         o.beta,
		BetaPassword: o.betaPassword,
	})

	if err := steamcmd.Run(ctx, commands...); err != nil {
		return nil, err
	}

	return xtar.Compress(installDir), nil
}
