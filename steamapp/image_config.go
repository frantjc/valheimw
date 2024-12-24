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

func ImageConfig(ctx context.Context, appID int, cfg *v1.Config, opts ...Opt) (*v1.Config, error) {
	o := &Opts{
		installDir:   "/",
		platformType: steamcmd.DefaultPlatformType,
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
		if strings.Contains(launch.Config.OSList, o.platformType.String()) {
			if o.launchType == "" || strings.EqualFold(launch.Type, o.launchType) {
				cfg.Labels["cc.frantj.sindri.id"] = fmt.Sprint(appID)
				if appInfo.Common != nil {
					cfg.Labels["cc.frantj.sindri.name"] = appInfo.Common.Name
					cfg.Labels["cc.frantj.sindri.type"] = appInfo.Common.Type
				}
				branchName := DefaultBranchName
				if o.beta != "" {
					branchName = o.beta
				}
				cfg.Labels["cc.frantj.sindri.branch"] = branchName
				if branch, ok := appInfo.Depots.Branches[branchName]; ok {
					cfg.Labels["cc.frantj.sindri.buildid"] = fmt.Sprint(branch.BuildID)
					cfg.Labels["cc.frantj.sindri.description"] = branch.Description
				}
				cfg.Entrypoint = []string{
					filepath.Join(o.installDir, launch.Executable),
				}
				cfg.Cmd = xslice.Filter(regexp.MustCompile(`\s+`).Split(launch.Arguments, -1), func(arg string, _ int) bool {
					return arg != ""
				})
				cfg.WorkingDir = o.installDir
				return cfg, nil
			}
		}
	}

	launchTypeAddendum := ""
	if o.launchType != "" {
		launchTypeAddendum = fmt.Sprintf("for launch type %s", o.launchType)
	}

	return nil, fmt.Errorf("app ID %d does not support %s, only %s%s", appInfo.Common.GameID, o.platformType, appInfo.Common.OSList, launchTypeAddendum)
}
