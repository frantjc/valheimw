package command

import (
	"runtime"

	"github.com/frantjc/sindri"
	"github.com/spf13/cobra"
)

func NewMist() *cobra.Command {
	var (
		cmd = &cobra.Command{
			Use:           "mist",
			Args:          cobra.ExactArgs(2),
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, args []string) error {
				return sindri.Extract(cmd.Context(), args[0], args[1])
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")

	return cmd
}
