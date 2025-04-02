package command

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetCommon(cmd *cobra.Command, version string) *cobra.Command {
	var verbosity int
	cmd.PersistentFlags().CountVarP(&verbosity, "verbose", "V", fmt.Sprintf("Verbosity for %s.", cmd.Name()))
	cmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		var (
			slog = slog.New(slog.NewTextHandler(cmd.OutOrStdout(), &slog.HandlerOptions{
				Level: slog.Level(int(slog.LevelError) - int(slog.LevelError-slog.LevelWarn)*verbosity),
			}))
			slogr = logr.FromSlogHandler(slog.Handler())
		)

		ctrl.SetLogger(slogr)
		cmd.SetContext(logr.NewContext(cmd.Context(), slogr))
	}

	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.Version = version
	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")

	return cmd
}
