package command

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
)

type SlogLeveler struct {
	level *slog.Level
}

func (s *SlogLeveler) init() {
	if s.level == nil {
		l := slog.LevelError
		if os.Getenv("DEBUG") != "" {
			l = slog.LevelDebug
		}
		s.level = &l
	}
}

func (s *SlogLeveler) Level() slog.Level {
	s.init()
	return *s.level
}

func (s *SlogLeveler) AddFlags(flags *pflag.FlagSet) {
	s.init()
	flags.AddFlag(&pflag.Flag{
		Name:      "debug",
		Shorthand: "d",
		Value: &Bool[slog.Level]{
			Value: s.level,
			IfSet: slog.LevelDebug,
		},
		NoOptDefVal: "true",
		Usage:       "Print debug logs",
	})
	flags.AddFlag(&pflag.Flag{
		Name:      "quiet",
		Shorthand: "q",
		Value: &Bool[slog.Level]{
			Value: s.level,
			IfSet: slog.LevelError,
		},
		NoOptDefVal: "true",
		Usage:       "Minimize logs",
	})
	flags.AddFlag(&pflag.Flag{
		Name:      "verbose",
		Shorthand: "v",
		Value: &Count[slog.Level]{
			Value:     s.level,
			Increment: slog.LevelWarn - slog.LevelError,
		},
		NoOptDefVal: "+1",
		Usage:       "More vebose logging",
	})
}

func newSlogHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return slog.New(slog.NewJSONHandler(w, opts)).Handler()
}

func newVersion(version string) *cobra.Command {
	cmd := SetCommon(&cobra.Command{
		Use: "version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		},
	}, "")

	return cmd
}

func SetCommon(cmd *cobra.Command, version string) *cobra.Command {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.Flags().BoolP("help", "h", false, "Help for "+cmd.Name())

	slogLeveler := new(SlogLeveler)
	slogLeveler.AddFlags(cmd.Flags())
	cmd.PreRun = func(cmd *cobra.Command, _ []string) {
		slogr := logr.FromSlogHandler(newSlogHandler(cmd.OutOrStdout(), &slog.HandlerOptions{
			Level: slogLeveler.Level(),
		}))

		cmd.SetContext(
			logr.NewContext(
				cmd.Context(),
				slogr,
			),
		)

		ctrl.SetLogger(slogr)
	}

	if version != "" {
		cmd.AddCommand(newVersion(version))
	}

	return cmd
}
