package steamcmd

import (
	"context"
	"fmt"
	"os/exec"
)

// IsInstalled checks whether `steamcmd` is installed
// and findable or not.
func IsInstalled() bool {
	bin, err := exec.LookPath("steamcmd")
	return bin != "" && err == nil
}

// NewCommand builds an *exec.Cmd to execute
// `steamcmd` with the given Commands.
func Run(ctx context.Context, cmds *Commands) (*exec.Cmd, error) {
	if !IsInstalled() {
		return nil, fmt.Errorf("steamcmd not installed")
	}

	//nolint:gosec
	return exec.CommandContext(ctx, "steamcmd", cmds.ToArgs()...), nil
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
