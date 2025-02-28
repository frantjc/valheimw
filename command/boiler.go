package command

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
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
		addr      int
		buildkitd string
		bucket    string
		db        string
		cmd       = &cobra.Command{
			Use: "boiler",
			RunE: func(cmd *cobra.Command, _ []string) error {
				var (
					eg, ctx  = errgroup.WithContext(cmd.Context())
					log      = logr.FromContextOrDiscard(ctx)
					registry = &steamapp.PullRegistry{
						ImageBuilder: &steamapp.ImageBuilder{},
					}
					srv = &http.Server{
						ReadHeaderTimeout: time.Second * 5,
						Handler:           contreg.NewPullHandler(registry),
						BaseContext: func(_ net.Listener) context.Context {
							return ctx
						},
					}
				)

				l, err := net.Listen("tcp", fmt.Sprintf(":%d", addr))
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

	cmd.Flags().IntVar(&addr, "addr", 5000, "Port for boiler to listen on.")
	cmd.Flags().StringVar(&buildkitd, "buildkitd", appdefaults.Address, "BuildKitd URL for boiler.")
	cmd.Flags().StringVar(&bucket, "bucket", fmt.Sprintf("file://%s?create_dir=1&no_tmp_dir=1", filepath.Join(cache.Dir, "boiler")), "Bucket URL for boiler.")
	cmd.Flags().StringVar(&db, "db", fmt.Sprintf("dummy://%s", steamapp.DefaultDir), "Database URL for boiler.")

	return cmd
}
