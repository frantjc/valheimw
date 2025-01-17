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

func NewImageConfig(ctx context.Context, appID int, cfg *v1.Config, opts ...Opt) (*v1.Config, error) {
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

	noLaunchType := len(o.launchTypes) == 0

	for _, launch := range appInfo.Config.Launch {
		if launch.Config != nil && strings.Contains(launch.Config.OSList, o.platformType.String()) {
			if noLaunchType || xslice.Some(o.launchTypes, func(launchType string, _ int) bool {
				return strings.EqualFold(launch.Type, launchType)
			}) {
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
				cfg.Cmd = regexp.MustCompile(`\s+`).Split(launch.Arguments, -1)
				cfg.WorkingDir = o.installDir
				return cfg, nil
			}
		}
	}

	launchTypeAddendum := ""
	if !noLaunchType {
		launchTypeAddendum = fmt.Sprintf(" for launch types %s", o.launchTypes)
	}

	return nil, fmt.Errorf("app ID %d does not support %s, only %s%s", appInfo.Common.GameID, o.platformType, appInfo.Common.OSList, launchTypeAddendum)
}
