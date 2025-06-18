package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/frantjc/sindri/command"
	"github.com/frantjc/sindri/steamapp"
	"github.com/frantjc/sindri/valheim"
	xerrors "github.com/frantjc/x/errors"
	xos "github.com/frantjc/x/os"
	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/kubepod"
	_ "github.com/moby/buildkit/client/connhelper/nerdctlcontainer"
	_ "github.com/moby/buildkit/client/connhelper/podmancontainer"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob"
)

func main() {
	var (
		ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		cmd       = command.SetCommon(command.NewBoiler(), SemVer())
	)

	err := xerrors.Ignore(cmd.ExecuteContext(ctx), context.Canceled)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	stop()
	xos.ExitFromError(err)
}

const (
	scheme = "dummy"
)

func init() {
	steamapp.RegisterDatabase(
		new(databaseURLOpener),
		scheme,
	)
}

type databaseURLOpener struct{}

func (d *databaseURLOpener) OpenDatabase(_ context.Context, u *url.URL) (steamapp.Database, error) {
	if u.Scheme != scheme {
		return nil, fmt.Errorf("invalid scheme %s, expected %s", u.Scheme, scheme)
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
