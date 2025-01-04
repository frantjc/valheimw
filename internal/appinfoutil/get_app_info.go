package appinfoutil

import (
	"context"
	"fmt"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/cache"
)

type GetAppInfoOpts struct {
	login steamcmd.Login
	store cache.Store
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

func WithStore(store cache.Store) GetAppInfoOpt {
	return func(o *GetAppInfoOpts) {
		o.store = store
	}
}

func GetAppInfo(ctx context.Context, appID int, opts ...GetAppInfoOpt) (*steamcmd.AppInfo, error) {
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

	if o.store != nil {
		appInfo := &steamcmd.AppInfo{}
		found, err := o.store.Get(fmt.Sprintf("appinfo::%d", appID), appInfo)
		if found {
			return appInfo, nil
		} else if err != nil {
			return nil, err
		}
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
		if o.store != nil {
			if err = o.store.Set(fmt.Sprintf("appinfo::%d", appID), appInfo); err != nil {
				return nil, err
			}
		}

		return appInfo, nil
	}
}
