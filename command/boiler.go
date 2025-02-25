package command

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/frantjc/sindri/contreg"
	"github.com/frantjc/sindri/internal/cache"
	"github.com/frantjc/sindri/steamapp"
	"github.com/go-logr/logr"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/appdefaults"
	"github.com/spf13/cobra"
	"gocloud.dev/blob"
	"golang.org/x/sync/errgroup"
)

func NewBoiler() *cobra.Command {
	var (
		verbosity int
		addr      string
		buildkitd string
		bucket    string
		db        string
		cmd       = &cobra.Command{
			Use:           "boiler",
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, _ []string) error {
				var (
					slog     = newSlogr(cmd, verbosity)
					eg, ctx  = errgroup.WithContext(logr.NewContextWithSlogLogger(cmd.Context(), slog))
					log      = logr.FromContextOrDiscard(ctx)
					registry = &steamapp.PullRegistry{
						ImageBuilder: &steamapp.ImageBuilder{},
					}
					srv = &http.Server{
						Addr:              addr,
						ReadHeaderTimeout: time.Second * 5,
						Handler:           contreg.NewPullHandler(registry),
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

				registry.Bucket, err = blob.OpenBucket(ctx, bucket)
				if err != nil {
					return err
				}

				registry.ImageBuilder.Client, err = client.New(ctx, buildkitd)
				if err != nil {
					return err
				}

				registry.Database, err = steamapp.OpenDatabase(ctx, db)
				if err != nil {
					return err
				}
				defer registry.Database.Close()

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
	cmd.Flags().StringVar(&buildkitd, "buildkitd", appdefaults.Address, "BuildKitd URL")
	cmd.Flags().StringVar(&bucket, "bucket", fmt.Sprintf("file://%s?create_dir=1&no_tmp_dir=1", filepath.Join(cache.Dir, "boiler")), "bucket URL")
	cmd.Flags().StringVar(&db, "db", fmt.Sprintf("dummy://%s", steamapp.DefaultDir), "database URL")

	return cmd
}
