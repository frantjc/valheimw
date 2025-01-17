package command

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri"
	"github.com/frantjc/sindri/internal/appinfoutil"
	"github.com/frantjc/sindri/steamapp"
	xslice "github.com/frantjc/x/slice"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
)

func NewBoil() *cobra.Command {
	var (
		output, rawRef, rawBaseImageRef string
		beta, betaPassword              string
		username, password              string
		platformType                    string
		launchType                      string
		dir                             string
		cmd                             = &cobra.Command{
			Use:           "boil",
			Version:       sindri.SemVer(),
			Args:          cobra.ExactArgs(1),
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, args []string) error {
				appID, err := strconv.Atoi(args[0])
				if err != nil {
					return err
				}

				var (
					imageW     = cmd.OutOrStdout()
					updateW    = cmd.ErrOrStderr()
					outputName = "stdout"
				)

				if !xslice.Includes([]string{"", "-"}, output) {
					var err error
					imageW, err = os.Create(output)
					if err != nil {
						return err
					}

					updateW = cmd.OutOrStdout()
					outputName = output
				}

				opts := []steamapp.ImageOpt{
					steamapp.WithBaseImageRef(rawBaseImageRef),
					steamapp.WithOpts(
						steamapp.WithLogin(username, password, ""),
						steamapp.WithBeta(beta, betaPassword),
						steamapp.WithInstallDir(dir),
						steamapp.WithLaunchTypes(launchType),
					),
				}
				if platformType != "" {
					opts = append(opts, steamapp.WithOpts(steamapp.WithPlatformType(steamcmd.PlatformType(platformType))))
				}

				if rawBaseImageRef == "" {
					fmt.Fprint(updateW, "Loading default base image...")

					baseImage, err := tarball.Image(func() (io.ReadCloser, error) {
						return io.NopCloser(bytes.NewReader(imageTar)), nil
					}, nil)
					if err != nil {
						return err
					}

					fmt.Fprintln(updateW, "DONE")

					opts = append(opts, steamapp.WithBaseImage(baseImage))
				}

				ctx := cmd.Context()

				fmt.Fprintf(updateW, "Layering Steam app %d onto image...", appID)

				image, err := steamapp.NewImage(ctx, appID, opts...)
				if err != nil {
					return err
				}

				fmt.Fprintln(updateW, "DONE")

				if rawRef == "" {
					fmt.Fprintf(updateW, "Getting Steam app %d info...", appID)

					appInfo, err := appinfoutil.GetAppInfo(ctx, appID, appinfoutil.WithLogin(username, password, ""))
					if err != nil {
						fmt.Fprintf(updateW, "FAILED\n")
						return err
					}

					fmt.Fprintf(updateW, "%s...DONE\n", appInfo.Common.Name)

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

				ref, err := name.ParseReference(rawRef)
				if err != nil {
					return err
				}

				updateC := make(chan v1.Update)
				go func() {
					var (
						preamble = fmt.Sprintf("\rWriting %s to %s...", ref, outputName)
						m, n     int
					)

					for update := range updateC {
						n, _ = fmt.Fprintf(updateW, "%s%d%% (%s / %s)", preamble, 100*update.Complete/update.Total, byteCount(update.Complete), byteCount(update.Total))
						if o := m - n; m-n > 0 {
							fmt.Fprint(updateW, strings.Repeat(" ", o))
						} else {
							m = n
						}
					}

					fmt.Fprintf(updateW, "%sDONE\n", preamble)
				}()

				if err := tarball.Write(ref, image, imageW, tarball.WithProgress(updateC)); err != nil {
					return err
				}
				close(updateC)

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
	cmd.Flags().StringVar(&launchType, "launchtype", "server", "Steam app launch type")

	cmd.Flags().StringVar(&username, "username", "", "Steam username")
	cmd.Flags().StringVar(&password, "password", "", "Steam password")

	return cmd
}

func byteCount(b int64) string {
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
