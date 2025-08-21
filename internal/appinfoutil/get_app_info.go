package appinfoutil

import (
	"context"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/logutil"
)

type GetAppInfoOpts struct {
	Login steamcmd.Login
}

func (o *GetAppInfoOpts) Apply(opts *GetAppInfoOpts) {
	if o != nil {
		if opts != nil {
			opts.Login = o.Login
		}
	}
}

type GetAppInfoOpt interface {
	Apply(*GetAppInfoOpts)
}

func WithLogin(username, password, steamGuardCode string) GetAppInfoOpt {
	return &GetAppInfoOpts{
		Login: steamcmd.Login{
			Username:       username,
			Password:       password,
			SteamGuardCode: steamGuardCode,
		},
	}
}

func GetAppInfo(ctx context.Context, appID int, opts ...GetAppInfoOpt) (*steamcmd.AppInfo, error) {
	log := logutil.SloggerFrom(ctx).With("appID", appID)

	if appInfo, found := steamcmd.GetAppInfo(appID); found {
		log.Debug("app info cached in memory")
		return appInfo, nil
	}

	var (
		o        = &GetAppInfoOpts{}
		errC     = make(chan error, 1)
		appInfoC = make(chan *steamcmd.AppInfo, 1)
	)
	defer close(errC)
	defer close(appInfoC)

	for _, opt := range opts {
		opt.Apply(o)
	}

	log.Debug("starting steamcmd to app_info_print")

	prompt, err := steamcmd.Start(ctx, o.Login, steamcmd.AppInfoRequest(appID))
	if err != nil {
		return nil, err
	}
	defer prompt.Close()

	go func() {
		for {
			appInfo, found := steamcmd.GetAppInfo(appID)
			if found {
				appInfoC <- appInfo
				return
			}

			if err = prompt.Run(ctx, steamcmd.AppInfoPrint(appID)); err != nil {
				errC <- err
				return
			}
		}
	}()

	select {
	case err := <-errC:
		return nil, err
	case appInfo := <-appInfoC:
		return appInfo, nil
	}
}
