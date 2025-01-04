package steamapp

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/appinfoutil"
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

	appInfo, err := appinfoutil.GetAppInfo(ctx, appID,
		appinfoutil.WithLogin(o.login.Username, o.login.Password, o.login.SteamGuardCode),
		appinfoutil.WithStore(o.store),
	)
	if err != nil {
		return nil, err
	}

	for _, launch := range appInfo.Config.Launch {
		if strings.Contains(launch.Config.OSList, o.platformType.String()) {
			if o.launchType == "" || strings.EqualFold(launch.Type, o.launchType) {
				if cfg.Labels == nil {
					cfg.Labels = map[string]string{}
				}
				cfg.Labels["cc.frantj.sindri.id"] = fmt.Sprint(appID)
				cfg.Labels["cc.frantj.sindri.name"] = appInfo.Common.Name
				cfg.Labels["cc.frantj.sindri.type"] = appInfo.Common.Type
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
