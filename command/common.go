package command

import (
	"io"
	"log/slog"
	"runtime"

	"github.com/frantjc/sindri/internal/appinfoutil"
	"github.com/frantjc/sindri/internal/logutil"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newSlogHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return slog.New(slog.NewJSONHandler(w, opts)).Handler()
}

func SetCommon(cmd *cobra.Command, version string) *cobra.Command {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.Flags().BoolP("help", "h", false, "Help for "+cmd.Name())
	cmd.Flags().Bool("version", false, "Version for "+cmd.Name())
	cmd.Version = version
	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")

	slogConfig := new(logutil.SlogConfig)
	slogConfig.AddFlags(cmd.Flags())
	cmd.PreRun = func(cmd *cobra.Command, _ []string) {
		handler := newSlogHandler(cmd.OutOrStdout(), &slog.HandlerOptions{
			Level: slogConfig,
		})
		cmd.SetContext(logutil.SloggerInto(cmd.Context(), slog.New(handler)))
		ctrl.SetLogger(logr.FromSlogHandler(handler))
	}
	cmd.PostRun = func(_ *cobra.Command, _ []string) {
		_ = appinfoutil.Close()
	}

	return cmd
}
