package command

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/adrg/xdg"
	"github.com/frantjc/sindri"
	"github.com/frantjc/sindri/internal/cache"
	"github.com/spf13/cobra"
)

func NewMist() *cobra.Command {
	var (
		clean bool
		cmd = &cobra.Command{
			Use:           "mist",
			Args:          cobra.ExactArgs(2),
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if clean {
					return errors.Join(
						os.RemoveAll(cache.Dir),
						os.RemoveAll(filepath.Join(xdg.CacheHome, "steamcmd")),
					)
				}

				return sindri.Extract(cmd.Context(), args[0], args[1])
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")

	cmd.Flags().BoolVar(&clean, "clean", false, "clean cache and exit")
	
	return cmd
}
