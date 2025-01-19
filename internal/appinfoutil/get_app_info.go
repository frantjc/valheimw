package appinfoutil

import (
	"context"

	"github.com/frantjc/go-steamcmd"
)

type GetAppInfoOpts struct {
	login steamcmd.Login
}

type GetAppInfoOpt func(*GetAppInfoOpts)

func WithLogin(username, password, steamGuardCode string) GetAppInfoOpt {
	return func(o *GetAppInfoOpts) {
		o.login = steamcmd.Login{
			Username:       username,
			Password:       password,
			SteamGuardCode: steamGuardCode,
		}
	}
}

func GetAppInfo(ctx context.Context, appID int, opts ...GetAppInfoOpt) (*steamcmd.AppInfo, error) {
	if appInfo, found := steamcmd.GetAppInfo(appID); found {
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
		opt(o)
	}

	prompt, err := steamcmd.Start(ctx, o.login, steamcmd.AppInfoRequest(appID))
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
