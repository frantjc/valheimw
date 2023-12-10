package command

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/frantjc/go-ingress"
	"github.com/frantjc/sindri"
	"github.com/frantjc/sindri/thunderstore"
	"github.com/frantjc/sindri/valheim"
	xtar "github.com/frantjc/sindri/x/tar"
	"github.com/spf13/cobra"
)

// NewSindri is the entrypoint for Sindri.
func NewSindri() *cobra.Command {
	var (
		verbosity int
		airgap    bool
		port      int64
		opts      = &valheim.Opts{
			Password: os.Getenv("VALHEIM_PASSWORD"),
		}
		root, state string
		mods        []string
		cmd         = &cobra.Command{
			Use:           "sindri",
			Version:       sindri.GetSemver(),
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
				thunderstoreURL, err := url.Parse("https://valheim.thunderstore.io/")
				if err != nil {
					return err
				}

				s, err := sindri.New(
					valheim.SteamAppID,
					valheim.BepInEx,
					thunderstore.NewClient(thunderstoreURL),
					sindri.WithRootDir(root),
					sindri.WithStateDir(state),
				)
				if err != nil {
					return err
				}

				if !airgap {
					log.Info("installing Valheim")

					if err = s.AppUpdate(ctx); err != nil {
						return err
					}

					for _, mod := range mods {
						log.Info("installing mod " + mod)

						if _, err = s.AddMod(ctx, mod); err != nil {
							return err
						}
					}
				}

				rc, err := s.Extract()
				if err != nil {
					return err
				}

				tmp, err := os.MkdirTemp(state, "")
				if err != nil {
					return err
				}

				if err = xtar.Extract(rc, tmp); err != nil {
					return err
				}

				if err = rc.Close(); err != nil {
					return err
				}

				opts.SaveDir = filepath.Join(root, "valheim")

				subCmd, err := valheim.NewCommand(ctx, tmp, opts)
				if err != nil {
					return err
				}
				sindri.LogExec(ctx, subCmd)

				addr := fmt.Sprintf(":%d", port)
				l, err := net.Listen("tcp", addr)
				if err != nil {
					return err
				}
				defer l.Close()

				var (
					errC          = make(chan error, 1)
					modTarHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						rc, err := s.ExtractMods()
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						defer rc.Close()

						w.Header().Add("Content-Type", "application/tar")

						_, _ = io.Copy(w, rc)
					})
					modTgzHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						rc, err := s.ExtractMods()
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						defer rc.Close()

						w.Header().Add("Content-Type", "application/gzip")

						gzw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
						if err != nil {
							gzw = gzip.NewWriter(w)
						}
						defer gzw.Close()

						_, _ = io.Copy(gzw, rc)
					})
					modHdrHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if accept := r.Header.Get("Accept"); strings.Contains(accept, "application/gzip") {
							modTgzHandler(w, r)
							return
						} else if strings.Contains(accept, "application/tar") {
							modTarHandler(w, r)
							return
						}

						w.WriteHeader(http.StatusNotAcceptable)
					})
					srv = &http.Server{
						Addr:              addr,
						ReadHeaderTimeout: time.Second * 5,
						BaseContext: func(_ net.Listener) context.Context {
							return ctx
						},
						Handler: ingress.New(
							ingress.ExactPath("/mods.tar", modTarHandler),
							ingress.ExactPath("/mods.gz", modTgzHandler),
							ingress.ExactPath("/mods.tgz", modTgzHandler),
							ingress.ExactPath("/mods.tar.gz", modTgzHandler),
							ingress.ExactPath("/mods", modHdrHandler),
						),
					}
				)

				go func() {
					log.Info("running Valheim")

					errC <- subCmd.Run()
				}()

				go func() {
					log.Info("listening on " + addr)

					errC <- srv.Serve(l)
				}()
				defer srv.Close()

				return <-errC
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")
	cmd.Flags().CountVarP(&verbosity, "verbose", "V", "verbosity for Sindri")

	cmd.Flags().StringVarP(&root, "root", "r", filepath.Join(xdg.CacheHome, "sindri/root"), "root directory for Sindri. Valheim savedir resides here")
	_ = cmd.MarkFlagDirname("root")

	cmd.Flags().StringVarP(&state, "state", "s", filepath.Join(xdg.CacheHome, "sindri/state"), "state directory for Sindri")
	_ = cmd.MarkFlagDirname("state")

	cmd.Flags().StringArrayVarP(&mods, "mod", "m", nil, "Thunderstore mods (case-sensitive)")

	cmd.Flags().Int64Var(&opts.Port, "valheim-port", 0, "port for Valheim (0 to use default)")
	cmd.Flags().StringVar(&opts.World, "valheim-world", "sindri", "world for Valheim")
	cmd.Flags().StringVar(&opts.Name, "valheim-name", "sindri", "name for Valheim")

	cmd.Flags().Int64Var(&port, "port", 8080, "port for Sindri")

	cmd.Flags().BoolVarP(&airgap, "airgap", "a", false, "do not redownload Valheim or mods")

	return cmd
}
