package appinfoutil

import (
	"context"
	"sync"

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

var (
	mu     sync.Mutex
	prompt *steamcmd.Prompt
)

func GetAppInfo(ctx context.Context, appID int, opts ...GetAppInfoOpt) (*steamcmd.AppInfo, error) {
	mu.Lock()
	defer mu.Unlock()

	var (
		_   = logutil.SloggerFrom(ctx).With("appID", appID)
		o   = &GetAppInfoOpts{}
		err error
	)

	for _, opt := range opts {
		opt.Apply(o)
	}

	if prompt == nil {
		if prompt, err = steamcmd.Start(ctx); err != nil {
			return nil, err
		}
	}

	if err = prompt.Run(ctx, o.Login, steamcmd.AppInfoRequest(appID), steamcmd.AppInfoPrint(appID)); err != nil {
		return nil, err
	}

	for {
		if appInfo, found := steamcmd.GetAppInfo(appID); found {
			return appInfo, nil
		}

		if err = prompt.Run(ctx, steamcmd.AppInfoPrint(appID)); err != nil {
			return nil, err
		}
	}
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if prompt != nil {
		return prompt.Close()
	}

	return nil
}
