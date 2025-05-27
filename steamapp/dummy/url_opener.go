package dummy

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/frantjc/sindri/steamapp"
	"github.com/frantjc/sindri/valheim"
)

const (
	Scheme = "dummy"
)

func init() {
	steamapp.RegisterDatabase(
		new(databaseURLOpener),
		Scheme,
	)
}

type databaseURLOpener struct{}

func (d *databaseURLOpener) OpenDatabase(_ context.Context, u *url.URL) (steamapp.Database, error) {
	if u.Scheme != Scheme {
		return nil, fmt.Errorf("invalid scheme %s, expected %s", u.Scheme, Scheme)
	}

	return &Database{u.Path}, nil
}

type Database struct {
	Dir string
}

var _ steamapp.Database = &Database{}

func (g *Database) GetBuildImageOpts(
	_ context.Context,
	appID int,
	branch string,
) (*steamapp.GettableBuildImageOpts, error) {
	switch appID {
	case valheim.SteamappID:
		opts := &steamapp.GettableBuildImageOpts{
			AptPkgs: []string{
				"ca-certificates",
			},
			LaunchType: "server",
			Execs: []string{
				fmt.Sprintf("rm -r %s %s %s %s",
					filepath.Join(g.Dir, "docker"),
					filepath.Join(g.Dir, "docker_start_server.sh"),
					filepath.Join(g.Dir, "start_server_xterm.sh"),
					filepath.Join(g.Dir, "start_server.sh"),
				),
				fmt.Sprintf("ln -s %s /usr/lib/x86_64-linux-gnu/steamclient.so",
					filepath.Join(g.Dir, "linux64/steamclient.so"),
				),
			},
			Entrypoint: []string{filepath.Join(g.Dir, "valheim_server.x86_64")},
		}

		if branch != "" && branch != steamapp.DefaultBranchName {
			opts.BetaPassword = "yesimadebackups"
		}

		return opts, nil
	case 1963720:
		// Core Keeper server.
		return &steamapp.GettableBuildImageOpts{
			AptPkgs: []string{
				"ca-certificates",
				"curl",
				"locales",
				"libxi6",
				"xvfb",
			},
			LaunchType: "server",
			Execs: []string{
				fmt.Sprintf("ln -s %s /usr/lib/x86_64-linux-gnu/steamclient.so",
					filepath.Join(g.Dir, "linux64/steamclient.so"),
				),
			},
			Entrypoint: []string{filepath.Join(g.Dir, "_launch.sh"), "-logfile", "/dev/stdout"},
		}, nil
	case 2394010:
		// Palworld server.
		return &steamapp.GettableBuildImageOpts{
			AptPkgs: []string{
				"ca-certificates",
				"xdg-user-dirs",
			},
			LaunchType: "default",
		}, nil
	case 1690800:
		return &steamapp.GettableBuildImageOpts{
			LaunchType: "default",
		}, nil
	}

	// Assume it works out of the box.
	return &steamapp.GettableBuildImageOpts{}, nil
}
