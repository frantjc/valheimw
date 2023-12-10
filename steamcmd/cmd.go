package steamcmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"

	xsyscall "github.com/frantjc/sindri/x/syscall"
)

// IsInstalled checks whether `steamcmd`` is installed
// and findable or not.
func IsInstalled() bool {
	bin, err := exec.LookPath("steamcmd")
	return bin != "" && err == nil
}

// Commands is a helper struct to build arguments
// to pass to `steamcmd`.
type Commands struct {
	ForceInstallDir string
	Login           string
	AppUpdate       string
	Validate        bool
	Beta            string
	BetaPassword    string
}

// ToArgs transforms Commands into an array
// of strings to pass to `steamcmd`.
func (c *Commands) ToArgs() []string {
	args := []string{}

	if c.ForceInstallDir != "" {
		args = append(args, "+force_install_dir", strings.TrimSpace(c.ForceInstallDir))
	}

	if c.Login != "" {
		args = append(args, "+login", strings.TrimSpace(c.Login))
	} else {
		args = append(args, "+login", "anonymous")
	}

	if c.AppUpdate != "" {
		args = append(args, "+app_update", strings.TrimSpace(c.AppUpdate))
	}

	if c.Beta != "" {
		args = append(args, "-beta", strings.TrimSpace(c.Beta))
	}

	if c.BetaPassword != "" {
		args = append(args, "-betapassword", strings.TrimSpace(c.BetaPassword))
	}

	if c.Validate {
		args = append(args, "-validate")
	}

	return append(args, "+quit")
}

var (
	// Username is the OS username to execute
	// `steamcmd` as (default current).
	Username = os.Getenv("STEAMCMD_USERNAME")
)

// NewCommand builds am *exec.Cmd to execute
// `steamcmd` with the given Commands.
func NewCommand(ctx context.Context, cmds *Commands) (*exec.Cmd, error) {
	if !IsInstalled() {
		return nil, fmt.Errorf("steamcmd not installed")
	}

	//nolint:gosec
	cmd := exec.CommandContext(ctx, "steamcmd", cmds.ToArgs()...)
	if Username != "" {
		if usr, err := user.Lookup(Username); err != nil {
			return nil, err
		} else if usr != nil {
			if cmd.SysProcAttr == nil {
				cmd.SysProcAttr = &syscall.SysProcAttr{}
			}

			if cmd.SysProcAttr.Credential, err = xsyscall.UserCredential(usr); err != nil {
				return nil, err
			}
		}
	}

	return cmd, nil
}

// TODO: implement this to use when not running in a
// container that already has steamcmd installed.
// func Install(ctx context.Context) error {
// 	steamQuestion := exec.CommandContext(ctx, "debconf-set-selections")
// 	steamQuestion.Stdin = strings.NewReader(`steam steam/question select "I AGREE"`)

// 	steamLicense := exec.CommandContext(ctx, "debconf-set-selections")
// 	steamLicense.Stdin = strings.NewReader("steam steam/license note ''")

// 	exec.CommandContext(ctx, "dpkg", "--add-architecture", "i386")
// 	exec.CommandContext(
// 		ctx, "apt-get",
// 		"update",
// 	)
// 	exec.CommandContext(
// 		ctx, "apt-get",
// 		"install",
// 		"-y",
// 		"--no-install-recommends",
// 		"lib32gcc1",
// 		"lib32stdc++6",
// 		"steamcmd",
// 	)
// 	exec.CommandContext(ctx, "steamcmd", "+quit")

// 	wd, err := os.Getwd()
// 	if err != nil {
// 		return err
// 	}

// 	if err = os.Link("/usr/games/steamcmd", filepath.Join(wd, "steamcmd")); err != nil {
// 		return err
// 	}

// 	// ln -s /usr/games/steamcmd /usr/bin/steamcmd
// 	// ln -s $HOME/.local/share/Steam/steamcmd/linux32 $HOME/.steam/sdk32
// 	// ln -s $HOME/.local/share/Steam/steamcmd/linux64 $HOME/.steam/sdk64
// 	// ln -s $HOME/.steam/sdk32/steamclient.so $HOME/.steam/sdk32/steamservice.so
// 	// ln -s $HOME/.steam/sdk64/steamclient.so $HOME/.steam/sdk64/steamservice.so
// }
