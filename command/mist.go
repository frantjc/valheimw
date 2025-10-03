package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/adrg/xdg"
	"github.com/frantjc/valheimw"
	"github.com/frantjc/valheimw/internal/cache"
	"github.com/spf13/cobra"
)

func NewMist() *cobra.Command {
	var (
		clean bool
		cmd   = &cobra.Command{
			Use: "mist",
			RunE: func(cmd *cobra.Command, args []string) error {
				if clean {
					return errors.Join(
						os.RemoveAll(cache.Dir),
						os.RemoveAll(filepath.Join(xdg.CacheHome, "steamcmd")),
					)
				}

				if lenArgs := len(args); lenArgs != 2 {
					return fmt.Errorf("accepts 2 arg(s), received %d", lenArgs)
				}

				return valheimw.Extract(cmd.Context(), args[0], args[1])
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")

	cmd.Flags().BoolVar(&clean, "clean", false, "Clean the cache and exit")

	return cmd
}
