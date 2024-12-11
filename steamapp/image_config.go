package steamapp

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/frantjc/go-steamcmd"
	xslice "github.com/frantjc/x/slice"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func ImageConfig(ctx context.Context, appID int, opts ...Opt) (*v1.Config, error) {
	o := &Opts{
		installDir: "/",
	}

	for _, opt := range opts {
		opt(o)
	}

	prompt, err := steamcmd.Start(ctx)
	if err != nil {
		return nil, err
	}
	defer prompt.Close(ctx)

	if err := prompt.Login(ctx, steamcmd.WithAccount(o.username, o.password)); err != nil {
		return nil, err
	}

	appInfo, err := prompt.AppInfoPrint(ctx, appID)
	if err != nil {
		return nil, err
	}

	if appInfo == nil || appInfo.Config == nil {
		return nil, fmt.Errorf("no app info config found")
	}

	for _, launch := range appInfo.Config.Launch {
		if strings.EqualFold(launch.Type, "server") && strings.Contains(launch.Config.OSList, o.platformType.String()) {
			return &v1.Config{
				Entrypoint: []string{
					filepath.Join(o.installDir, launch.Executable),
				},
				Cmd: xslice.Filter(regexp.MustCompile(`\s+`).Split(launch.Arguments, -1), func(arg string, _ int) bool {
					return arg != ""
				}),
				WorkingDir: o.installDir,
			}, nil
		}
	}

	return nil, fmt.Errorf("app ID %d does not support %s, only %s", appInfo.Common.GameID, o.platformType, appInfo.Common.OSList)
}
