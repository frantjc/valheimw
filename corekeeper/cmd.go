package corekeeper

import (
	"context"
	"os/exec"
	"path/filepath"
)

// NewCommand builds an *exec.Cmd for the Corekeeper executable
// in the given directory with the given options.
func NewCommand(ctx context.Context, dir string) (*exec.Cmd, error) {
	var (
		//nolint:gosec
		cmd = exec.CommandContext(
			ctx,
			filepath.Join(dir, "_launch.sh"),
			"-logfile", "/dev/stdout",
		)
	)

	cmd.Dir = dir

	return cmd, nil
}
