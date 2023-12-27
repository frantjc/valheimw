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
