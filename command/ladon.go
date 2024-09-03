package command

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/adrg/xdg"
	"github.com/frantjc/sindri"
	"github.com/frantjc/sindri/corekeeper"
	xtar "github.com/frantjc/x/archive/tar"
	"github.com/spf13/cobra"
)

// NewLadon is the entrypoint for `ladon`.
func NewLadon() *cobra.Command {
	var (
		noDownload bool
		beta, betaPassword   string
		root, state          string
		verbosity            int
		cmd = &cobra.Command{
			Use:           "ladon",
			Version:       sindri.SemVer(),
			SilenceErrors: true,
			SilenceUsage:  true,
			PreRun: func(cmd *cobra.Command, _ []string) {
				cmd.SetContext(
					sindri.WithLogger(cmd.Context(), sindri.NewLogger().V(2-verbosity)),
				)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				var (
					ctx = cmd.Context()
					log = sindri.LoggerFrom(ctx)
				)
				state, err := filepath.Abs(state)
				if err != nil {
					return err
				}

				s, err := sindri.New(
					nil, nil,
					sindri.WithRootDir(root),
					sindri.WithStateDir(state),
					sindri.WithBeta(beta, betaPassword),
				)
				if err != nil {
					return err
				}
	
				if !noDownload {
					if err = s.AppUpdate(ctx, corekeeper.SteamAppID); err != nil {
						return err
					}
				}

				runDir, err := os.MkdirTemp(state, "")
				if err != nil {
					return err
				}

				log.Info("installing Core Keeper to " + runDir)

				corekeeperTar, err := s.Extract([]string{corekeeper.SteamAppID})
				if err != nil {
					return err
				}

				if err = xtar.Extract(tar.NewReader(corekeeperTar), runDir); err != nil {
					return err
				}

				errC := make(chan error)

				subCmd, err := corekeeper.NewCommand(ctx, runDir)
				if err != nil {
					return err
				}
				sindri.LogExec(log, subCmd)
				defer os.RemoveAll(runDir)

				go func() {
					log.Info("running Core Keeper in " + runDir)

					if err := subCmd.Run(); err != nil {
						errC <- fmt.Errorf("core keeper: %w", err)
					} else {
						errC <- nil
					}
				}()

				return <-errC
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")
	cmd.Flags().CountVarP(&verbosity, "verbose", "V", "verbosity for sindri")

	cmd.Flags().StringVarP(&root, "root", "r", filepath.Join(xdg.DataHome, "sindri"), "root directory for sindri (-savedir resides here)")
	_ = cmd.MarkFlagDirname("root")

	cmd.Flags().StringVarP(&state, "state", "s", filepath.Join(xdg.RuntimeDir, "sindri"), "state directory for sindri")
	_ = cmd.MarkFlagDirname("state")

	cmd.Flags().BoolVar(&noDownload, "no-download", false, "do not redownload Valheim or mods")

	cmd.Flags().StringVar(&beta, "beta", "", "Steam beta branch")
	cmd.Flags().StringVar(&betaPassword, "beta-password", "", "Steam beta password")

	return cmd
}

