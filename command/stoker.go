package command

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/frantjc/sindri/internal/api"
	"github.com/frantjc/sindri/steamapp/postgres"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func NewStoker() *cobra.Command {
	var (
		addr int
		path string
		db   string
		cmd  = &cobra.Command{
			Use: "stoker",
			RunE: func(cmd *cobra.Command, _ []string) error {
				var (
					eg, ctx = errgroup.WithContext(cmd.Context())
					_       = logr.FromContextOrDiscard(ctx)
					srv     = &http.Server{
						ReadHeaderTimeout: time.Second * 5,
						BaseContext: func(_ net.Listener) context.Context {
							return ctx
						},
					}
				)

				u, err := url.Parse(db)
				if err != nil {
					return err
				}

				database, err := postgres.NewDatabase(ctx, u)
				if err != nil {
					return err
				}
				defer database.Close()

				srv.Handler = api.NewHandler(path, database)

				l, err := net.Listen("tcp", fmt.Sprintf(":%d", addr))
				if err != nil {
					return err
				}
				defer l.Close()

				eg.Go(func() error {
					return srv.Serve(l)
				})

				eg.Go(func() error {
					<-ctx.Done()
					cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second*30)
					defer cancel()
					if err = srv.Shutdown(cctx); err != nil {
						return err
					}
					return ctx.Err()
				})

				return eg.Wait()
			},
		}
	)

	cmd.Flags().IntVarP(&addr, "addr", "a", 5050, "Port for stoker to listen on")
	cmd.Flags().StringVar(&db, "db", "postgres://localhost:5432?sslmode=disable", "Database URL for stoker")
	cmd.Flags().StringVar(&path, "path", "/", "Base path for stoker")

	return cmd
}
