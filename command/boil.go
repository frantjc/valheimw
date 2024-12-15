package command

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/contreg"
	"github.com/frantjc/sindri/internal/layerutil"
	"github.com/frantjc/sindri/steamapp"
	xslice "github.com/frantjc/x/slice"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
)

func NewBoil() *cobra.Command {
	var (
		output, rawRef, rawBaseImageRef string
		beta, betaPassword              string
		username, password              string
		platformType                    string
		dir                             string
		cmd                             = &cobra.Command{
			Use:           "boil",
			Args:          cobra.ExactArgs(1),
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, args []string) error {
				var (
					imageW  = cmd.OutOrStdout()
					updateW = cmd.ErrOrStderr()
				)

				if !xslice.Includes([]string{"", "-"}, output) {
					var err error
					imageW, err = os.Create(output)
					if err != nil {
						return err
					}

					updateW = cmd.OutOrStdout()
				}

				appID, err := strconv.Atoi(args[0])
				if err != nil {
					return err
				}

				var (
					ctx     = cmd.Context()
					updateC = make(chan v1.Update)
				)
				go func() {
					var (
						byteCount = func(b int64) string {
							const unit = 1000
							if b < unit {
								return fmt.Sprintf("%db", b)
							}
							div, exp := int64(unit), 0
							for n := b / unit; n >= unit; n /= unit {
								div *= unit
								exp++
							}
							return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "kmgt"[exp])
						}
					)

					for update := range updateC {
						_, _ = fmt.Fprintf(updateW, "\r(%s / %s) %d%%", byteCount(update.Complete), byteCount(update.Total), 100*update.Complete/update.Total)
					}

					fmt.Fprintln(updateW, " DONE")
				}()
				defer close(updateC)

				var (
					opts = []steamapp.Opt{
						steamapp.WithAccount(username, password),
						steamapp.WithBeta(beta, betaPassword),
					}
					image = empty.Image
				)

				if platformType != "" {
					opts = append(opts, steamapp.WithPlatformType(steamcmd.PlatformType(platformType)))
				}

				if rawBaseImageRef != "" {
					baseImageRef, err := name.ParseReference(rawBaseImageRef)
					if err != nil {
						return err
					}

					image, err = contreg.DefaultClient.Pull(ctx, baseImageRef)
					if err != nil {
						return err
					}
				}

				if rawRef == "" {
					prompt, err := steamcmd.Start(ctx)
					if err != nil {
						return err
					}

					if err := prompt.Login(ctx, steamcmd.WithAccount(username, password)); err != nil {
						return err
					}

					appInfo, err := prompt.AppInfoPrint(ctx, appID)
					if err != nil {
						return err
					}

					if err = prompt.Close(ctx); err != nil {
						return err
					}

					branchName := steamapp.DefaultBranchName
					if beta != "" {
						branchName = beta
					}

					rawRef = fmt.Sprintf(
						"boil.frantj.cc/%d:%s",
						appInfo.Common.GameID,
						branchName,
					)
				}

				cfgf, err := image.ConfigFile()
				if err != nil {
					return err
				}

				cfg, err := steamapp.ImageConfig(ctx, appID, &cfgf.Config, append(opts, steamapp.WithInstallDir(dir))...)
				if err != nil {
					return err
				}

				image, err = mutate.Config(image, *cfg)
				if err != nil {
					return err
				}

				layer, err := layerutil.ReproducibleBuildLayerInDirFromOpener(
					func() (io.ReadCloser, error) {
						return steamapp.Open(
							ctx,
							appID,
							opts...,
						)
					},
					dir,
					"", "",
				)
				if err != nil {
					return err
				}

				image, err = mutate.AppendLayers(image, layer)
				if err != nil {
					return err
				}

				ref, err := name.ParseReference(rawRef)
				if err != nil {
					return err
				}

				if err := tarball.Write(ref, image, imageW, tarball.WithProgress(updateC)); err != nil {
					return err
				}

				return nil
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")

	cmd.Flags().StringVarP(&output, "output", "o", "", "file to write the image to (default stdout)")
	cmd.Flags().StringVarP(&rawRef, "ref", "r", "", "ref to write the image as (default boil.frantj.cc/<steamappid>:<branch>)")
	cmd.Flags().StringVarP(&rawBaseImageRef, "base", "b", "", "base image to build upon (default scratch)")

	cmd.Flags().StringVar(&beta, "beta", "", "Steam beta branch")
	cmd.Flags().StringVar(&betaPassword, "beta-password", "", "Steam beta password")

	cmd.Flags().StringVar(&dir, "dir", "/home/boil/steamapp", "Steam app install directory")

	cmd.Flags().StringVar(&platformType, "platformtype", "", "Steam app platform type")

	cmd.Flags().StringVar(&username, "username", "", "Steam username")
	cmd.Flags().StringVar(&password, "password", "", "Steam password")

	return cmd
}
