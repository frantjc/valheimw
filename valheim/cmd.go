package valheim

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	xos "github.com/frantjc/x/os"
)

var (
	//go:embed start_server_sindri.sh
	startServerBepInExSh []byte
)

// NewCommand builds an *exec.Cmd for the Valheim executable
// in the given directory with the given options.
func NewCommand(ctx context.Context, dir string, opts *Opts) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "windows", "darwin":
		return nil, fmt.Errorf("%s incompatible with BepInEx", runtime.GOOS)
	}

	if strings.Contains(opts.World, opts.Password) || len(opts.Password) < 5 {
		return nil, fmt.Errorf("-password must be >=5 characters and not contained within the world name")
	}

	if !filepath.IsAbs(dir) {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
	}

	var (
		//nolint:gosec
		cmd = exec.CommandContext(
			ctx,
			filepath.Join(dir, "valheim_server.x86_64"),
			append(
				opts.ToArgs(),
				// Unclear if these do anything or where I got them,
				// but once upon a time I was lead to believe that
				// they improve performance.
				"-batchmode",
				"-nographics",
				"-screen-width", "640",
				"-screen-height", "480",
				"-screen-quality", "Fastest",
			)...,
		)
		ldLibraryPath = xos.JoinPath(
			os.Getenv("LD_LIBRARY_PATH"),
			filepath.Join(cmd.Dir, "linux64"),
		)
	)

	cmd.Dir = dir
	cmd.Env = append(
		os.Environ(),
		"SteamAppId=892970",
	)

	if opts.BepInEx {
		// We seem to have to use a script because
		// a shell does something special with one
		// of the env vars that we are using.
		cmd.Path = filepath.Join(dir, "start_server_sindri.sh")

		f, err := os.OpenFile(cmd.Path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if _, err := f.Write(startServerBepInExSh); err != nil {
			return nil, err
		}

		cancel := cmd.Cancel
		cmd.Cancel = func() error {
			defer os.Remove(f.Name())
			return cancel()
		}

		cmd.Env = append(
			cmd.Env,
			"DOORSTOP_ENABLE=TRUE",
			fmt.Sprintf("DOORSTOP_INVOKE_DLL_PATH=%s", filepath.Join(cmd.Dir, "BepInEx/core/BepInEx.Preloader.dll")),
			fmt.Sprintf("LD_PRELOAD=%s",
				xos.JoinPath(
					"libdoorstop_x64.so",
					os.Getenv("LD_PRELOAD"),
				),
			),
		)

		ldLibraryPath = xos.JoinPath(filepath.Join(cmd.Dir, "doorstop_libs"), ldLibraryPath)
	}

	cmd.Env = append(
		cmd.Env,
		fmt.Sprintf("LD_LIBRARY_PATH=%s",
			ldLibraryPath,
		),
	)

	return cmd, nil
}
