package command

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/img"
	"github.com/frantjc/sindri/steamapp"
	xos "github.com/frantjc/x/os"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/moby/term"
	"github.com/spf13/cobra"
)

func NewZoroark() *cobra.Command {
	var (
		username, password string
		beta, betaPassword string
		platformType       string
		rawTag             string
		cmd                = &cobra.Command{
			Use:           "zoroark",
			Args:          cobra.ExactArgs(1),
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, args []string) error {
				appID, err := strconv.Atoi(args[0])
				if err != nil {
					return err
				}

				cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
				if err != nil {
					return err
				}

				ctx := cmd.Context()

				if rawTag == "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Getting Steam app %d info...", appID)

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

					fmt.Fprintf(cmd.OutOrStdout(), "%s...DONE\n", appInfo.Common.Name)

					branchName := steamapp.DefaultBranchName
					if beta != "" {
						branchName = beta
					}

					rawTag = fmt.Sprintf(
						"boil.frantj.cc/%d:%s",
						appInfo.Common.GameID,
						branchName,
					)
				}

				tag, err := name.NewTag(rawTag)
				if err != nil {
					return err
				}

				if _, _, err := cli.ImageInspectWithRaw(ctx, tag.String()); err != nil {
					fmt.Fprint(cmd.OutOrStdout(), "Loading default base image...")

					baseImage, err := tarball.Image(func() (io.ReadCloser, error) {
						return io.NopCloser(bytes.NewReader(imageTar)), nil
					}, nil)
					if err != nil {
						return err
					}

					fmt.Fprintln(cmd.OutOrStdout(), "DONE")

					fmt.Fprintf(cmd.OutOrStdout(), "Layering Steam app %d onto image...", appID)

					image, err := img.SteamappImage(ctx, appID,
						img.WithBaseImage(baseImage),
						img.WithUser("root", "root"),
						img.WithSteamappOpts(
							steamapp.WithAccount(username, password),
							steamapp.WithBeta(beta, betaPassword),
							steamapp.WithInstallDir("/steamapp"),
							steamapp.WithPlatformType(steamcmd.PlatformType(platformType)),
						),
					)
					if err != nil {
						return err
					}

					fmt.Fprintln(cmd.OutOrStdout(), "DONE")

					updateC := make(chan v1.Update)
					go func() {
						var (
							preamble = fmt.Sprintf("\rWriting %s to %s...", tag, cli.DaemonHost())
							m, n     int
						)

						for update := range updateC {
							n, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s%d%% (%s / %s)", preamble, 100*update.Complete/update.Total, byteCount(update.Complete), byteCount(update.Total))
							if o := m - n; m-n > 0 {
								fmt.Fprint(cmd.OutOrStdout(), strings.Repeat(" ", o))
							} else {
								m = n
								n = 0
							}
						}

						fmt.Fprintf(cmd.OutOrStdout(), "%sDONE\n", preamble)
					}()

					if _, err = daemon.Write(tag, image, daemon.WithContext(ctx), daemon.WithClient(cli)); err != nil {
						return err
					}
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Creating Steam app %d container...", appID)

				cr, err := cli.ContainerCreate(ctx,
					&container.Config{
						Image:        tag.String(),
						AttachStdout: true,
						AttachStderr: true,
						Env: []string{
							"DISPLAY=host.docker.internal:0",
						},
					},
					&container.HostConfig{
						Binds: []string{
							"/tmp/.X11-unix:/tmp/.X11-unix",
						},
						Resources: container.Resources{
							DeviceRequests: []container.DeviceRequest{
								{
									Count:        -1,
									Capabilities: [][]string{{"gpu"}},
								},
							},
						},
						// AutoRemove: true,
					},
					nil, nil,
					fmt.Sprint(appID),
				)
				if err != nil {
					return err
				}

				fmt.Fprintln(cmd.OutOrStdout(), "DONE")

				fmt.Fprintf(cmd.OutOrStdout(), "Attaching to Steam app container %s...", cr.ID)

				hjr, err := cli.ContainerAttach(ctx, cr.ID, container.AttachOptions{
					Stream: true,
					Stdout: true,
					Stderr: true,
					Logs:   true,
				})
				if err != nil {
					return err
				}

				var (
					errC = make(chan error, 1)
					outC = make(chan any, 1)
				)
				go func() {
					defer hjr.Close()
					defer close(outC)
					defer close(errC)
					if _, err = stdcopy.StdCopy(
						cmd.OutOrStdout(),
						cmd.ErrOrStderr(),
						hjr.Reader,
					); err != nil {
						errC <- err
					}
				}()

				fmt.Fprintln(cmd.OutOrStdout(), "DONE")

				if err = cli.ContainerStart(ctx, cr.ID, container.StartOptions{}); err != nil {
					return err
				}

				select {
				case err = <-errC:
					if _, ok := err.(term.EscapeError); ok {
						err = nil
					}
				case <-ctx.Done():
					err = ctx.Err()
				case <-outC:
				}
				if err != nil {
					return err
				}

				cei, err := cli.ContainerExecInspect(ctx, cr.ID)
				if err != nil {
					return err
				}

				if cei.ExitCode > 0 {
					return xos.NewExitCodeError(fmt.Errorf("container exited with nonzero exit code"), cei.ExitCode)
				}

				return nil
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")

	cmd.Flags().StringVar(&username, "username", "", "Steam username")
	cmd.Flags().StringVar(&password, "password", "", "Steam password")

	cmd.Flags().StringVar(&beta, "beta", "", "Steam beta branch")
	cmd.Flags().StringVar(&betaPassword, "beta-password", "", "Steam beta password")

	cmd.Flags().StringVar(&platformType, "platformtype", steamcmd.PlatformTypeLinux.String(), "Steam app platform type")

	return cmd
}
