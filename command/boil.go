package command

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/contreg"
	"github.com/frantjc/sindri/steamapp"
	xslice "github.com/frantjc/x/slice"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
)

func NewBoil() *cobra.Command {
	var (
		output, rawRef, rawBaseImageRef    string
		beta, betaPassword                 string
		username, password string
		cmd                                = &cobra.Command{
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
					for update := range updateC {
						_, _ = fmt.Fprintf(updateW, "%d/%d\n", update.Complete, update.Total)
					}
				}()
				defer close(updateC)

				var (
					opts = []steamapp.Opt{
						steamapp.WithAccount(username, password),
						steamapp.WithBeta(beta, betaPassword),
					}
					image = empty.Image
				)

				if rawBaseImageRef != "" {
					baseImageRef, err := name.ParseReference(rawBaseImageRef)
					if err != nil {
						return err
					}

					image, err = contreg.DefaultClient.Read(ctx, baseImageRef)
					if err != nil {
						return err
					}
				}

				if rawRef == "" {
					prompt, err := steamcmd.Start(ctx)
					if err != nil {
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
						"sindri.frantj.cc/%d:%d",
						appInfo.Common.GameID,
						appInfo.Depots.Branches[branchName].BuildID,
					)
				}

				cfg, err := image.ConfigFile()
				if err != nil {
					return err
				}

				newCfg, err := steamapp.ImageConfig(ctx, appID, opts...)
				if err != nil {
					return err
				}

				cfg.Config.Entrypoint = newCfg.Entrypoint
				cfg.Config.Cmd = newCfg.Cmd
				cfg.Config.WorkingDir = newCfg.WorkingDir

				image, err = mutate.Config(image, cfg.Config)
				if err != nil {
					return err
				}

				layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
					return steamapp.Open(
						ctx,
						appID,
						steamapp.WithBeta(beta, betaPassword),
						steamapp.WithAccount(username, password),
					)
				})
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

	cmd.Flags().StringVarP(&output, "output", "o", "", "file to write the image to (default stdout)")
	cmd.Flags().StringVarP(&rawRef, "ref", "r", "", "ref to write the image as (default sindri.frantj.cc/<steamappid>:<steamappbuildid>)")
	cmd.Flags().StringVarP(&rawBaseImageRef, "base", "b", "", "base image to build upon (default scratch)")

	cmd.Flags().StringVar(&beta, "beta", "", "Steam beta branch")
	cmd.Flags().StringVar(&betaPassword, "beta-password", "", "Steam beta password")

	cmd.Flags().StringVar(&username, "username", "", "Steam username")
	cmd.Flags().StringVar(&password, "password", "", "Steam password")

	return cmd
}
