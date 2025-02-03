package command

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/frantjc/go-ingress"
	"github.com/frantjc/sindri"
	"github.com/frantjc/sindri/internal/cache"
	"github.com/frantjc/sindri/steamapp"
	"github.com/frantjc/sindri/thunderstore"
	"github.com/frantjc/sindri/valheim"
	xtar "github.com/frantjc/x/archive/tar"
	"github.com/go-logr/logr"
	"github.com/mmatczuk/anyflag"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

const (
	bepInExNamespace = "denikson"
	bepInExName      = "BepInExPack_Valheim"
)

func newSlogr(cmd *cobra.Command, verbosity int) *slog.Logger {
	return slog.New(slog.NewTextHandler(cmd.OutOrStdout(), &slog.HandlerOptions{
		Level: slog.LevelError - (slog.LevelError-slog.LevelWarn)*slog.Level(verbosity),
	}))
}

func NewValheimw() *cobra.Command {
	var (
		addr     string
		openOpts = &steamapp.OpenOpts{
			LaunchType: "server",
		}
		mods        []string
		verbosity   int
		noDB, noFWL bool
		playerLists = &valheim.PlayerLists{}
		opts        = &valheim.Opts{
			Password: os.Getenv("VALHEIM_PASSWORD"),
		}
		cmd = &cobra.Command{
			Use:           "valheimw",
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, _ []string) error {
				wd := filepath.Join(cache.Dir, "valheimw")
				defer os.RemoveAll(wd)

				if err := os.MkdirAll(wd, 0777); err != nil {
					return err
				}

				var (
					ctx            = logr.NewContextWithSlogLogger(cmd.Context(), newSlogr(cmd, verbosity))
					log            = logr.FromContextOrDiscard(ctx)
					eg, installCtx = errgroup.WithContext(ctx)
				)

				if len(mods) > 0 {
					log.Info("resolving dependency tree")

					pkgs, err := thunderstore.DependencyTree(ctx, mods...)
					if err != nil {
						return err
					}

					for _, pkg := range pkgs {
						dir := fmt.Sprintf("BepInEx/plugins/%s", pkg.String())

						if pkg.Namespace == bepInExNamespace && pkg.Name == bepInExName {
							opts.BepInEx = true
							dir = ""
						}

						log.Info("installing package", "package", pkg.String(), "rel", dir)

						eg.Go(func() error {
							return sindri.Extract(installCtx,
								fmt.Sprintf("%s://%s", thunderstore.Scheme, pkg.String()),
								filepath.Join(wd, dir),
							)
						})
					}

					if !opts.BepInEx && len(pkgs) > 0 {
						opts.BepInEx = true

						log.Info("installing bepinex as nothing else requested it")

						eg.Go(func() error {
							return sindri.Extract(installCtx,
								fmt.Sprintf("%s://%s-%s", thunderstore.Scheme, bepInExNamespace, bepInExName),
								wd,
							)
						})
					}
				}

				log.Info("installing Valheim server", "id", valheim.SteamappID)

				eg.Go(func() error {
					return sindri.Extract(installCtx,
						fmt.Sprintf("%s://%d?%s", steamapp.Scheme, valheim.SteamappID, steamapp.URLValues(openOpts).Encode()),
						wd,
					)
				})

				if err := eg.Wait(); err != nil {
					return err
				}

				log.Info("finished installing")
				log.Info("configuring HTTP server")

				var (
					zHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write([]byte("ok\n"))
					})
					paths = []ingress.Path{
						ingress.ExactPath("/readyz", zHandler),
						ingress.ExactPath("/livez", zHandler),
						ingress.ExactPath("/healthz", zHandler),
					}
				)

				if !noDB {
					log.Info("exposing .db-related endpoints")

					var (
						dbHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							w.Header().Add("Content-Disposition", "attachment")

							db, err := valheim.OpenDB(opts.SaveDir, opts.World)
							if err != nil {
								w.WriteHeader(http.StatusInternalServerError)
								return
							}
							defer db.Close()

							_, _ = io.Copy(w, db)
						})
					)

					paths = append(paths,
						ingress.ExactPath("/world.db", dbHandler),
						ingress.ExactPath(filepath.Join("/", opts.World+".db"), dbHandler),
					)
				}

				if !noFWL {
					log.Info("exposing .fwl-related endpoints")

					valheimMapURL, err := url.Parse("https://valheim-map.world?offset=0,0&zoom=0.600&view=0&ver=0.217.22")
					if err != nil {
						return err
					}

					var (
						seedJSONHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							seed, err := valheim.ReadWorldSeed(opts.SaveDir, opts.World)
							if err != nil {
								w.WriteHeader(http.StatusInternalServerError)
								return
							}

							w.Header().Add("Content-Type", "application/json")

							_, _ = w.Write([]byte(`{"seed":"` + seed + `"}`))
						})
						seedTxtHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							seed, err := valheim.ReadWorldSeed(opts.SaveDir, opts.World)
							if err != nil {
								w.WriteHeader(http.StatusInternalServerError)
								return
							}

							w.Header().Add("Content-Type", "text/plain")

							_, _ = w.Write([]byte(seed))
						})
						seedHdrHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							if accept := r.Header.Get("Accept"); strings.Contains(accept, "application/json") {
								seedJSONHandler(w, r)
								return
							}

							seedTxtHandler(w, r)
						})
						mapHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							seed, err := valheim.ReadWorldSeed(opts.SaveDir, opts.World)
							if err != nil {
								w.WriteHeader(http.StatusInternalServerError)
								return
							}

							q := valheimMapURL.Query()
							q.Set("seed", seed)
							valheimMapURL.RawQuery = q.Encode()

							http.Redirect(w, r, valheimMapURL.String(), http.StatusTemporaryRedirect)
						})
						fwlHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							w.Header().Add("Content-Disposition", "attachment")

							fwl, err := valheim.OpenFWL(opts.SaveDir, opts.World)
							if err != nil {
								w.WriteHeader(http.StatusInternalServerError)
								return
							}
							defer fwl.Close()

							_, _ = io.Copy(w, fwl)
						})
					)

					paths = append(paths,
						ingress.ExactPath("/seed.json", seedJSONHandler),
						ingress.ExactPath("/seed.txt", seedTxtHandler),
						ingress.ExactPath("/seed", seedHdrHandler),
						ingress.ExactPath("/map", mapHandler),
						ingress.ExactPath("/world.fwl", fwlHandler),
						ingress.ExactPath(filepath.Join("/", opts.World+".fwl"), fwlHandler),
					)
				}

				if !noDB && !noFWL {
					var (
						worldsTarHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							w.Header().Add("Content-Type", "application/tar")
							w.Header().Add("Content-Disposition", "attachment")

							_, _ = io.Copy(w, xtar.Compress(filepath.Join(opts.SaveDir, "worlds_local")))
						})
						worldsTgzHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							gzw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
							if err != nil {
								gzw = gzip.NewWriter(w)
							}
							defer gzw.Close()

							w.Header().Add("Content-Type", "application/gzip")
							w.Header().Add("Content-Disposition", "attachment")

							_, _ = io.Copy(gzw, xtar.Compress(filepath.Join(opts.SaveDir, "worlds_local")))
						})
						worldsHdrHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							if accept := r.Header.Get("Accept"); strings.Contains(accept, "application/gzip") {
								w.Header().Add("Content-Disposition", "filename=file worlds.tar.gz")
								worldsTgzHandler(w, r)
								return
							} else if strings.Contains(accept, "application/tar") {
								w.Header().Add("Content-Disposition", "filename=file worlds.tar")
								worldsTarHandler(w, r)
								return
							}

							w.WriteHeader(http.StatusNotAcceptable)
						})
					)

					paths = append(paths,
						ingress.ExactPath("/worlds.tar", worldsTarHandler),
						ingress.ExactPath("/worlds.tar.gz", worldsTgzHandler),
						ingress.ExactPath("/worlds.tgz", worldsTgzHandler),
						ingress.ExactPath("/worlds_local.tar.gz", worldsTgzHandler),
						ingress.ExactPath("/worlds_local.tgz", worldsTgzHandler),
						ingress.ExactPath("/worlds", worldsHdrHandler),
						ingress.ExactPath("/worlds_local", worldsHdrHandler),
					)
				}

				if len(mods) > 0 {
					log.Info("exposing mod-related endpoints")

					var (
						modTarHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							tw := tar.NewWriter(w)
							defer tw.Close()

							w.Header().Add("Content-Type", "application/tar")
							w.Header().Add("Content-Disposition", "attachment")

							for _, mod := range mods {
								rc, err := sindri.Open(ctx, mod)
								if err != nil {
									w.WriteHeader(http.StatusInternalServerError)
									return
								}
								defer rc.Close()

								tr := tar.NewReader(rc)

								for {
									hdr, err := tr.Next()
									if errors.Is(err, io.EOF) {
										break
									} else if err != nil {
										w.WriteHeader(http.StatusInternalServerError)
										return
									}

									if err = tw.WriteHeader(hdr); err != nil {
										w.WriteHeader(http.StatusInternalServerError)
										return
									}

									//nolint:gosec
									if _, err = io.Copy(tw, tr); err != nil {
										w.WriteHeader(http.StatusInternalServerError)
										return
									}
								}
							}
						})
						modTgzHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							gzw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
							if err != nil {
								gzw = gzip.NewWriter(w)
							}
							defer gzw.Close()

							tw := tar.NewWriter(gzw)
							defer tw.Close()

							w.Header().Add("Content-Type", "application/gzip")
							w.Header().Add("Content-Disposition", "attachment")

							for _, mod := range mods {
								rc, err := sindri.Open(ctx, mod)
								if err != nil {
									w.WriteHeader(http.StatusInternalServerError)
									return
								}
								defer rc.Close()

								tr := tar.NewReader(rc)

								for {
									hdr, err := tr.Next()
									if errors.Is(err, io.EOF) {
										break
									} else if err != nil {
										w.WriteHeader(http.StatusInternalServerError)
										return
									}

									if err = tw.WriteHeader(hdr); err != nil {
										w.WriteHeader(http.StatusInternalServerError)
										return
									}

									//nolint:gosec
									if _, err = io.Copy(tw, tr); err != nil {
										w.WriteHeader(http.StatusInternalServerError)
										return
									}
								}
							}
						})
						modHdrHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							if accept := r.Header.Get("Accept"); strings.Contains(accept, "application/gzip") {
								w.Header().Add("Content-Disposition", "filename=file mods.tar.gz")
								modTgzHandler(w, r)
								return
							} else if strings.Contains(accept, "application/tar") {
								w.Header().Add("Content-Disposition", "filename=file mods.tar")
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
				}

				eg, egctx := errgroup.WithContext(ctx)

				sub, err := valheim.NewCommand(egctx, wd, opts)
				if err != nil {
					return err
				}

				sub.Stdin = cmd.InOrStdin()
				sub.Stdout = cmd.OutOrStdout()
				sub.Stderr = cmd.ErrOrStderr()

				log.Info("starting Valheim server")

				eg.Go(sub.Run)

				l, err := net.Listen("tcp", addr)
				if err != nil {
					return err
				}
				defer l.Close()

				srv := &http.Server{
					Addr:              addr,
					ReadHeaderTimeout: time.Second * 5,
					Handler:           ingress.New(paths...),
				}

				eg.Go(func() error {
					log.Info("listening...", "addr", l.Addr().String())

					return srv.Serve(l)
				})

				eg.Go(func() error {
					<-egctx.Done()
					cctx, cancel := context.WithTimeout(context.WithoutCancel(egctx), time.Second*30)
					defer cancel()
					if err = srv.Shutdown(cctx); err != nil {
						return err
					}
					return egctx.Err()
				})

				return eg.Wait()
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")
	cmd.Flags().CountVarP(&verbosity, "verbose", "V", "verbosity")

	cmd.Flags().StringArrayVarP(&mods, "mod", "m", nil, "Thunderstore mods (case-sensitive)")

	cmd.Flags().BoolVar(&noDB, "no-db", false, "do not expose the world .db file for download")
	cmd.Flags().BoolVar(&noFWL, "no-fwl", false, "do not expose the world .fwl file information")

	cmd.Flags().StringVar(&addr, "addr", ":8080", "address")

	cmd.Flags().StringVar(&opts.SaveDir, "savedir", filepath.Join(cache.Dir, "valheim"), "Valheim server -savedir")
	cmd.Flags().StringVar(&opts.Name, "name", "sindri", "Valheim server -name")
	cmd.Flags().Int64Var(&opts.Port, "port", 0, "Valheim server -port (0 to use default)")
	cmd.Flags().StringVar(&opts.World, "world", "sindri", "Valheim server -world")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Valheim server make -public")

	cmd.Flags().DurationVar(&opts.SaveInterval, "save-interval", 0, "Valheim server -saveinterval duration")
	cmd.Flags().Int64Var(&opts.Backups, "backups", 0, "Valheim server -backup amount")
	cmd.Flags().DurationVar(&opts.BackupShort, "backup-short", 0, "Valheim server -backupshort duration")
	cmd.Flags().DurationVar(&opts.BackupLong, "backup-long", 0, "Valheim server -backuplong duration")

	cmd.Flags().BoolVar(&opts.Crossplay, "crossplay", false, "Valheim server enable -crossplay")

	cmd.Flags().StringVar(&opts.InstanceID, "instance-id", "", "Valheim server -instanceid")

	cmd.Flags().Var(
		anyflag.NewValue(
			"",
			&opts.Preset,
			anyflag.EnumParser(
				valheim.PresetCasual,
				valheim.PresetEasy,
				valheim.PresetNormal,
				valheim.PresetHard,
				valheim.PresetHardcore,
				valheim.PresetImmersive,
				valheim.PresetHammer,
			),
		),
		"preset",
		"Valheim server -preset",
	)

	cmd.Flags().Var(
		anyflag.NewValue(
			"",
			&opts.CombatModifier,
			anyflag.EnumParser(
				valheim.CombatModifierVeryEasy,
				valheim.CombatModifierEasy,
				valheim.CombatModifierHard,
				valheim.CombatModifierVeryHard,
			),
		),
		"combat-modifier",
		"Valheim server -modifier combat",
	)

	cmd.Flags().Var(
		anyflag.NewValue(
			"",
			&opts.DeathPenaltyModifier,
			anyflag.EnumParser(
				valheim.DeathPenaltyModifierCasual,
				valheim.DeathPenaltyModifierVeryEasy,
				valheim.DeathPenaltyModifierEasy,
				valheim.DeathPenaltyModifierHard,
				valheim.DeathPenaltyModifierHardcore,
			),
		),
		"death-penalty-modifier",
		"Valheim server -modifier deathpenalty",
	)

	cmd.Flags().Var(
		anyflag.NewValue(
			"",
			&opts.ResourceModifier,
			anyflag.EnumParser(
				valheim.ResourceModifierMuchLess,
				valheim.ResourceModifierLess,
				valheim.ResourceModifierMore,
				valheim.ResourceModifierMuchMore,
				valheim.ResourceModifierMost,
			),
		),
		"resource-modifier",
		"Valheim server -modifier resources",
	)

	cmd.Flags().Var(
		anyflag.NewValue(
			"",
			&opts.RaidModifier,
			anyflag.EnumParser(
				valheim.RaidModifierNone,
				valheim.RaidModifierMuchLess,
				valheim.RaidModifierLess,
				valheim.RaidModifierMore,
				valheim.RaidModifierMuchMore,
			),
		),
		"raid-modifier",
		"Valheim server -modifier raids",
	)

	cmd.Flags().Var(
		anyflag.NewValue(
			"",
			&opts.PortalModifier,
			anyflag.EnumParser(
				valheim.PortalModifierCasual,
				valheim.PortalModifierHard,
				valheim.PortalModifierVeryHard,
			),
		),
		"portal-modifier",
		"Valheim server -modifier portals",
	)

	cmd.Flags().BoolVar(&opts.NoBuildCost, "no-build-cost", false, "Valheim server -setkey nobuildcost")
	cmd.Flags().BoolVar(&opts.PlayerEvents, "player-events", false, "Valheim server -setkey playerevents")
	cmd.Flags().BoolVar(&opts.PassiveMobs, "passive-mobs", false, "Valheim server -setkey passivemobs")
	cmd.Flags().BoolVar(&opts.NoMap, "no-map", false, "Valheim server -setkey nomap")

	cmd.Flags().Int64SliceVar(&playerLists.AdminIDs, "admin", nil, "Valheim server admin Steam IDs")
	cmd.Flags().Int64SliceVar(&playerLists.BannedIDs, "ban", nil, "Valheim server banned Steam IDs")
	cmd.Flags().Int64SliceVar(&playerLists.PermittedIDs, "permit", nil, "Valheim server permitted Steam IDs")

	cmd.Flags().StringVar(&openOpts.Beta, "beta", "", "Steam beta branch")
	cmd.Flags().StringVar(&openOpts.BetaPassword, "beta-password", "", "Steam beta password")

	return cmd
}
