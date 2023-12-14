package command

import (
	"compress/gzip"
	"context"
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
	"github.com/frantjc/sindri/internal/clienthelper"
	"github.com/frantjc/sindri/thunderstore"
	"github.com/frantjc/sindri/valheim"
	xtar "github.com/frantjc/sindri/x/tar"
	"github.com/spf13/cobra"
)

// NewSindri is the entrypoint for Sindri.
func NewSindri() *cobra.Command {
	var (
		addr               string
		airgap, modsOnly   bool
		beta, betaPassword string
		mods, rmMods       []string
		root, state        string
		verbosity          int
		playerLists        = &valheim.PlayerLists{}
		opts               = &valheim.Opts{
			Password: os.Getenv("VALHEIM_PASSWORD"),
		}
		cmd = &cobra.Command{
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
					thunderstore.NewClient(thunderstoreURL, thunderstore.WithDir(state)),
					sindri.WithRootDir(root),
					sindri.WithStateDir(state),
					sindri.WithBeta(beta, betaPassword),
				)
				if err != nil {
					return err
				}

				if !airgap {
					if len(mods) > 0 {
						// Mods first because they're going to be smaller
						// most of the time so it makes the whole process
						// a bit faster.
						log.Info("downloading mods " + strings.Join(append(mods, s.BepInEx.Fullname()), ", "))

						if err = s.AddMods(ctx, mods...); err != nil {
							return err
						}
					}

					if !modsOnly {
						log.Info("downloading Valheim")

						if err = s.AppUpdate(ctx); err != nil {
							return err
						}
					}
				}

				if err = s.RemoveMods(ctx, rmMods...); err != nil {
					return err
				}

				moddedValheimTar, err := s.Extract()
				if err != nil {
					return err
				}
				defer moddedValheimTar.Close()

				runDir, err := os.MkdirTemp(state, "")
				if err != nil {
					return err
				}

				if err = xtar.Extract(moddedValheimTar, runDir); err != nil {
					return err
				}

				if err = moddedValheimTar.Close(); err != nil {
					return err
				}

				opts.SaveDir = root

				if err := valheim.WritePlayerLists(opts.SaveDir, playerLists); err != nil {
					return err
				}

				errC := make(chan error, 1)

				subCmd, err := valheim.NewCommand(ctx, runDir, opts)
				if err != nil {
					return err
				}
				sindri.LogExec(log, subCmd)

				go func() {
					log.Info("running Valheim")

					errC <- subCmd.Run()
				}()

				var (
					seedJSONHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						seed, err := valheim.ReadSeed(opts)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						_, _ = w.Write([]byte(`{"seed":"` + seed + `"}`))
					})
					seedTxtHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						seed, err := valheim.ReadSeed(opts)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						_, _ = w.Write([]byte(seed))
					})
					seedHdrHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if accept := r.Header.Get("Accept"); strings.Contains(accept, "application/json") {
							seedJSONHandler(w, r)
						} else if strings.Contains(accept, "text/plain") {
							seedTxtHandler(w, r)
						}

						w.WriteHeader(http.StatusNotAcceptable)
					})
					paths = []ingress.Path{
						ingress.ExactPath("/seed.json", seedJSONHandler),
						ingress.ExactPath("/seed.txt", seedTxtHandler),
						ingress.ExactPath("/seed", seedHdrHandler),
					}
				)

				if packages, err := s.Mods(); len(packages) > 0 || err != nil {
					var (
						modTarHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							rc, err := s.ExtractMods()
							if err != nil {
								w.WriteHeader(http.StatusInternalServerError)
								return
							}
							defer rc.Close()

							w.Header().Add("Content-Type", "application/tar")

							if _, err = io.Copy(w, clienthelper.NewTarPrefixReader(r)); err == nil {
								_, _ = io.Copy(w, rc)
							}
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

							if _, err = io.Copy(gzw, clienthelper.NewTarPrefixReader(r)); err == nil {
								_, _ = io.Copy(gzw, rc)
							}
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
					)

					paths = append(paths,
						ingress.ExactPath("/mods.tar", modTarHandler),
						ingress.ExactPath("/mods.gz", modTgzHandler),
						ingress.ExactPath("/mods.tgz", modTgzHandler),
						ingress.ExactPath("/mods.tar.gz", modTgzHandler),
						ingress.ExactPath("/mods", modHdrHandler),
					)
				} else {
					log.Info("no mods, not serving mod download endpoints")
				}

				srv := &http.Server{
					Addr:              addr,
					ReadHeaderTimeout: time.Second * 5,
					BaseContext: func(_ net.Listener) context.Context {
						return ctx
					},
					Handler: ingress.New(paths...),
				}

				l, err := net.Listen("tcp", addr)
				if err != nil {
					return err
				}
				defer l.Close()

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

	cmd.Flags().StringVarP(&root, "root", "r", filepath.Join(xdg.DataHome, "sindri"), "root directory for Sindri. Valheim savedir resides here")
	_ = cmd.MarkFlagDirname("root")

	cmd.Flags().StringVarP(&state, "state", "s", filepath.Join(xdg.RuntimeDir, "sindri"), "state directory for Sindri")
	_ = cmd.MarkFlagDirname("state")

	cmd.Flags().StringArrayVarP(&mods, "mod", "m", nil, "Thunderstore mods (case-sensitive)")
	cmd.Flags().StringArrayVar(&rmMods, "rm", nil, "Thunderstore mods to remove (case-sensitive)")
	cmd.Flags().BoolVar(&modsOnly, "mods-only", false, "do not redownload Valheim")
	cmd.Flags().BoolVar(&airgap, "airgap", false, "do not redownload Valheim or mods")

	cmd.Flags().StringVar(&addr, "addr", ":8080", "address for Sindri")

	cmd.Flags().Int64Var(&opts.Port, "port", 0, "port for Valheim (0 to use default)")
	cmd.Flags().StringVar(&opts.World, "world", "sindri", "world for Valheim")
	cmd.Flags().StringVar(&opts.Name, "name", "sindri", "name for Valheim")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "make Valheim server public")

	cmd.Flags().IntSliceVar(&playerLists.Admins, "admin", nil, "Valheim server admin Steam IDs")
	cmd.Flags().IntSliceVar(&playerLists.Banned, "ban", nil, "Valheim server banned Steam IDs")
	cmd.Flags().IntSliceVar(&playerLists.Permitted, "permit", nil, "Valheim server permitted Steam IDs")

	cmd.Flags().StringVar(&beta, "beta", "", "Steam beta branch")
	cmd.Flags().StringVar(&betaPassword, "beta-password", "", "Steam beta password")

	return cmd
}
