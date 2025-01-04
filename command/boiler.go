package command

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/frantjc/sindri"
	"github.com/frantjc/sindri/distrib"
	"github.com/frantjc/sindri/internal/cache"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

//go:embed image.tar
var imageTar []byte

func NewBoiler() *cobra.Command {
	var (
		verbosity int
		addr      string
		registry  = &distrib.SteamappPuller{
			Dir:   "/home/boil/steamapp",
			User:  "boil",
			Group: "boil",
		}
		cmd = &cobra.Command{
			Use:           "boiler",
			Version:       sindri.SemVer(),
			Args:          cobra.MaximumNArgs(1),
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, args []string) error {
				var (
					slog    = newSlogr(cmd, verbosity)
					eg, ctx = errgroup.WithContext(logr.NewContextWithSlogLogger(cmd.Context(), slog))
					log     = logr.FromContextOrDiscard(ctx)
					srv     = &http.Server{
						Addr:              addr,
						ReadHeaderTimeout: time.Second * 5,
						Handler:           distrib.Handler(registry),
						BaseContext: func(_ net.Listener) context.Context {
							return logr.NewContextWithSlogLogger(context.Background(), slog)
						},
					}
				)

				l, err := net.Listen("tcp", addr)
				if err != nil {
					return err
				}
				defer l.Close()

				registry.Base, err = tarball.Image(func() (io.ReadCloser, error) {
					return io.NopCloser(bytes.NewReader(imageTar)), nil
				}, nil)
				if err != nil {
					return err
				}

				if len(args) > 0 {
					log.Info("using cache", "url", args[0])

					registry.Store, err = cache.NewStore(args[0])
					if err != nil {
						return err
					}
				}

				eg.Go(func() error {
					<-ctx.Done()
					cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second*30)
					defer cancel()
					if err = srv.Shutdown(cctx); err != nil {
						return err
					}
					return ctx.Err()
				})

				eg.Go(func() error {
					log.Info("listening...", "addr", l.Addr().String())

					return srv.Serve(l)
				})

				return eg.Wait()
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")
	cmd.Flags().CountVarP(&verbosity, "verbose", "V", "verbosity")

	cmd.Flags().StringVar(&addr, "addr", ":5000", "address")

	cmd.Flags().StringVar(&registry.Username, "username", "", "Steam username")
	cmd.Flags().StringVar(&registry.Password, "password", "", "Steam password")

	return cmd
}
