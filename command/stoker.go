package command

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/frantjc/sindri/internal/stokerhttp"
	"github.com/frantjc/sindri/steamapp/postgres"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func NewStoker() *cobra.Command {
	var (
		addr int
		db   string
		opts = &stokerhttp.Opts{
			Swagger: true,
		}
		cmd = &cobra.Command{
			Use: "stoker",
			RunE: func(cmd *cobra.Command, args []string) error {
				var (
					eg, ctx = errgroup.WithContext(cmd.Context())
					log     = logr.FromContextOrDiscard(ctx)
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

				if len(args) > 0 {
					var ex *exec.Cmd
					opts.Fallback, ex, err = stokerhttp.NewExecHandlerWithPortFromEnv(ctx, args[0], args[1:]...)
					if err != nil {
						return err
					}

					// A rough algorithm for making the working directory of
					// the exec the directory of the entrypoint in the case
					// of the args being something like `node /app/server.js`.
					for _, entrypoint := range args[1:] {
						if fi, err := os.Stat(entrypoint); err == nil {
							if fi.IsDir() {
								ex.Dir = filepath.Clean(entrypoint)
							} else {
								ex.Dir = filepath.Dir(entrypoint)
							}
							break
						}
					}

					log.Info("running exec fallback server")

					eg.Go(ex.Run)
				}

				l, err := net.Listen("tcp", fmt.Sprintf(":%d", addr))
				if err != nil {
					return err
				}
				defer l.Close()

				srv := &http.Server{
					ReadHeaderTimeout: time.Second * 5,
					Handler:           stokerhttp.NewAPIHandler(database, opts),
					BaseContext: func(_ net.Listener) context.Context {
						return cmd.Context()
					},
				}

				eg.Go(func() error {
					return srv.Serve(l)
				})

				eg.Go(func() error {
					<-ctx.Done()
					if err = srv.Shutdown(context.WithoutCancel(ctx)); err != nil {
						return err
					}
					return ctx.Err()
				})

				return eg.Wait()
			},
		}
	)

	cmd.Flags().IntVarP(&addr, "addr", "a", 5050, "Port for stoker to listen on")
	cmd.Flags().StringVarP(&opts.Path, "path", "p", "", "Base URL path for stoker")
	cmd.Flags().StringVar(&db, "db", "postgres://localhost:5432?sslmode=disable", "Database URL for stoker")

	return cmd
}
