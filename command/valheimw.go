package command

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/frantjc/go-ingress"
	"github.com/frantjc/valheimw"
	"github.com/frantjc/valheimw/internal/cache"
	"github.com/frantjc/valheimw/internal/logutil"
	"github.com/frantjc/valheimw/steamapp"
	"github.com/frantjc/valheimw/thunderstore"
	"github.com/frantjc/valheimw/valheim"
	xtar "github.com/frantjc/x/archive/tar"
	xslices "github.com/frantjc/x/slices"
	"github.com/mmatczuk/anyflag"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

const (
	bepInExNamespace = "denikson"
	bepInExName      = "BepInExPack_Valheim"
)

func NewValheimw() *cobra.Command {
	var (
		addr     int
		openOpts = &steamapp.OpenOpts{
			LaunchType: "server",
		}
		mods                   []string
		noDB, noFWL, noValheim bool
		playerLists            = &valheim.PlayerLists{}
		opts                   = &valheim.Opts{
			Password: os.Getenv("VALHEIM_PASSWORD"),
		}
		valheimMapWorldVersion string
		cmd                    = &cobra.Command{
			Use: "valheimw",
			RunE: func(cmd *cobra.Command, _ []string) error {
				wd := filepath.Join(cache.Dir, "valheimw")
				defer os.RemoveAll(wd)

				if err := os.MkdirAll(wd, 0775); err != nil {
					return err
				}

				var (
					ctx    = cmd.Context()
					log    = logutil.SloggerFrom(ctx)
					modded = len(mods) > 0
				)

				pkgs, err := thunderstore.DependencyTree(ctx, mods...)
				if err != nil {
					return err
				}

				if !noValheim {
					eg, installCtx := errgroup.WithContext(ctx)

					if modded {
						log.Info("resolving dependency tree")

						for _, pkg := range pkgs {
							dir := fmt.Sprintf("BepInEx/plugins/%s", pkg.String())
							isBepInEx := pkg.Namespace == bepInExNamespace && pkg.Name == bepInExName

							if isBepInEx {
								opts.BepInEx = true
								dir = "."
							} else if !xslices.Some(pkg.CommunityListings, func(communityListing thunderstore.CommunityListing, _ int) bool {
								return slices.Contains(communityListing.Categories, "Server-side")
							}) {
								continue
							}

							log.Info("installing package", "pkg", pkg.String(), "rel", dir)

							eg.Go(func() error {
								return valheimw.Extract(installCtx,
									fmt.Sprintf("%s://%s", thunderstore.Scheme, pkg.String()),
									filepath.Join(wd, dir),
								)
							})
						}

						if !opts.BepInEx {
							opts.BepInEx = true

							pkg, err := thunderstore.NewClient().GetPackage(ctx, &thunderstore.Package{
								Namespace: bepInExNamespace,
								Name:      bepInExName,
							})
							if err != nil {
								return err
							}

							pkgs = append(pkgs, *pkg)

							log.Info("installing latest BepInEx: no mods depended on a specific version", "pkg", pkg.String())

							eg.Go(func() error {
								return valheimw.Extract(installCtx,
									fmt.Sprintf("%s://%s", thunderstore.Scheme, pkg.String()),
									wd,
								)
							})
						}
					}

					log.Info("installing Valheim server")

					eg.Go(func() error {
						return valheimw.Extract(installCtx,
							fmt.Sprintf("%s://%d?%s", steamapp.Scheme, valheim.SteamappID, steamapp.URLValues(openOpts).Encode()),
							wd,
						)
					})

					if err := eg.Wait(); err != nil {
						return fmt.Errorf("installing game files: %w", err)
					}

					log.Info("finished installing")

					if modded {
						var (
							saveCfgDir    = filepath.Join(opts.SaveDir, "config")
							bepInExCfgDir = filepath.Join(wd, "BepInEx/config")
						)

						if err := os.MkdirAll(saveCfgDir, 0775); err != nil {
							return err
						}

						if err := xtar.Extract(
							tar.NewReader(xtar.Compress(saveCfgDir)),
							bepInExCfgDir,
						); err != nil {
							return err
						}

						defer func() {
							_ = xtar.Extract(
								tar.NewReader(xtar.Compress(bepInExCfgDir)),
								saveCfgDir,
							)
						}()
					}

					if err := valheim.WritePlayerLists(opts.SaveDir, playerLists); err != nil {
						return fmt.Errorf("writing player lists: %w", err)
					}
				}

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
								http.Error(w, err.Error(), http.StatusInternalServerError)
								return
							}
							defer db.Close()

							_, _ = io.Copy(w, db)
						})
					)

					paths = append(paths,
						ingress.ExactPath("/world.db", dbHandler),
						ingress.ExactPath(path.Join("/", fmt.Sprintf("%s.db", opts.World)), dbHandler),
					)
				}

				if !noFWL {
					log.Info("exposing .fwl-related endpoints")

					valheimMapURL, err := url.Parse(
						fmt.Sprintf(
							"https://valheim-map.world?offset=0,0&zoom=0.600&view=0&ver=%s",
							valheimMapWorldVersion,
						),
					)
					if err != nil {
						return err
					}

					var (
						seedJSONHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							seed, err := valheim.ReadWorldSeed(opts.SaveDir, opts.World)
							if err != nil {
								http.Error(w, err.Error(), http.StatusInternalServerError)
								return
							}

							w.Header().Add("Content-Type", "application/json")

							_, _ = w.Write([]byte(fmt.Sprintf(`{"seed":"%s"}`, seed)))
						})
						seedTxtHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							seed, err := valheim.ReadWorldSeed(opts.SaveDir, opts.World)
							if err != nil {
								http.Error(w, err.Error(), http.StatusInternalServerError)
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
								http.Error(w, err.Error(), http.StatusInternalServerError)
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
								http.Error(w, err.Error(), http.StatusInternalServerError)
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
						ingress.ExactPath(path.Join("/", fmt.Sprintf("%s.fwl", opts.World)), fwlHandler),
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

							w.Header().Add("Content-Type", "application/tar")
							w.Header().Add("Content-Encoding", "gzip")
							w.Header().Add("Content-Disposition", "attachment")

							_, _ = io.Copy(gzw, xtar.Compress(filepath.Join(opts.SaveDir, "worlds_local")))
						})
						worldsHdrHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							if accept := r.Header.Get("Accept"); strings.Contains(accept, "application/tar") {
								if acceptEncoding := r.Header.Get("Accept-Encoding"); strings.Contains(acceptEncoding, "gzip") {
									w.Header().Add("Content-Disposition", "filename=file worlds.tar.gz")
									worldsTgzHandler(w, r)
									return
								}

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
						ingress.ExactPath("/worlds_local.tar", worldsTarHandler),
						ingress.ExactPath("/worlds_local.tar.gz", worldsTgzHandler),
						ingress.ExactPath("/worlds_local.tgz", worldsTgzHandler),
						ingress.ExactPath("/worlds", worldsHdrHandler),
						ingress.ExactPath("/worlds_local", worldsHdrHandler),
					)
				}

				if modded {
					log.Info("exposing mod-related endpoints")

					var (
						modTarHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							tw := tar.NewWriter(w)
							defer tw.Close()

							w.Header().Add("Content-Type", "application/tar")
							w.Header().Add("Content-Disposition", "attachment")

							for _, pkg := range pkgs {
								if pkg.Namespace == bepInExNamespace && pkg.Name == bepInExName {
									continue
								} else if xslices.Every(pkg.CommunityListings, func(communityListing thunderstore.CommunityListing, _ int) bool {
									return !slices.Contains(communityListing.Categories, "Server-side")
								}) {
									continue
								}

								rc, err := valheimw.Open(ctx, fmt.Sprintf("%s://%s", thunderstore.Scheme, pkg.String()))
								if err != nil {
									http.Error(w, err.Error(), http.StatusInternalServerError)
									return
								}
								defer rc.Close()

								tr := tar.NewReader(rc)

								for {
									hdr, err := tr.Next()
									if errors.Is(err, io.EOF) {
										break
									} else if err != nil {
										http.Error(w, err.Error(), http.StatusInternalServerError)
										return
									}

									base := path.Base(hdr.Name)

									if ext := path.Ext(base); ext != ".dll" {
										continue
									} else {
										hdr.Name = path.Join("BepInEx/plugins", pkg.String(), base)
									}

									if err = tw.WriteHeader(hdr); err != nil {
										http.Error(w, err.Error(), http.StatusInternalServerError)
										return
									}

									//nolint:gosec
									if _, err = io.Copy(tw, tr); err != nil {
										http.Error(w, err.Error(), http.StatusInternalServerError)
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

							w.Header().Add("Content-Type", "application/tar")
							w.Header().Add("Content-Encoding", "application/gzip")
							w.Header().Add("Content-Disposition", "attachment")

							for _, pkg := range pkgs {
								if pkg.Namespace == bepInExNamespace && pkg.Name == bepInExName {
									continue
								} else if !xslices.Some(pkg.CommunityListings, func(communityListing thunderstore.CommunityListing, _ int) bool {
									return slices.Contains(communityListing.Categories, "Client-side")
								}) {
									continue
								}

								rc, err := valheimw.Open(ctx, fmt.Sprintf("%s://%s", thunderstore.Scheme, pkg.String()))
								if err != nil {
									http.Error(w, err.Error(), http.StatusInternalServerError)
									return
								}
								defer rc.Close()

								tr := tar.NewReader(rc)

								for {
									hdr, err := tr.Next()
									if errors.Is(err, io.EOF) {
										break
									} else if err != nil {
										http.Error(w, err.Error(), http.StatusInternalServerError)
										return
									}

									base := path.Base(hdr.Name)

									if ext := path.Ext(base); ext != ".dll" {
										continue
									} else {
										hdr.Name = path.Join("BepInEx/plugins", pkg.String(), base)
									}

									if err = tw.WriteHeader(hdr); err != nil {
										http.Error(w, err.Error(), http.StatusInternalServerError)
										return
									}

									//nolint:gosec
									if _, err = io.Copy(tw, tr); err != nil {
										http.Error(w, err.Error(), http.StatusInternalServerError)
										return
									}
								}
							}
						})
						modHdrHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							if accept := r.Header.Get("Accept"); strings.Contains(accept, "application/tar") {
								if acceptEncoding := r.Header.Get("Accept-Encoding"); strings.Contains(acceptEncoding, "gzip") {
									w.Header().Add("Content-Disposition", "filename=file mods.tar.gz")
									modTgzHandler(w, r)
								}
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

				if !noValheim {
					sub, err := valheim.NewCommand(egctx, wd, opts)
					if err != nil {
						return err
					}

					sub.Stdin = cmd.InOrStdin()
					sub.Stdout = cmd.OutOrStdout()
					sub.Stderr = cmd.ErrOrStderr()

					log.Info("starting Valheim server")

					eg.Go(sub.Run)
				}

				l, err := net.Listen("tcp", fmt.Sprintf(":%d", addr))
				if err != nil {
					return err
				}
				defer l.Close()

				srv := &http.Server{
					ReadHeaderTimeout: time.Second * 5,
					Handler:           ingress.New(paths...),
					BaseContext: func(_ net.Listener) context.Context {
						return cmd.Context()
					},
				}

				eg.Go(func() error {
					log.Info("listening...", "addr", l.Addr().String())

					return srv.Serve(l)
				})

				eg.Go(func() error {
					<-egctx.Done()
					if err = srv.Shutdown(context.WithoutCancel(egctx)); err != nil {
						return err
					}
					return egctx.Err()
				})

				return eg.Wait()
			},
		}
	)

	cmd.Flags().StringArrayVarP(&mods, "mod", "m", nil, "Thunderstore mods (case-sensitive)")

	cmd.Flags().BoolVar(&noDB, "no-db", false, "Do not expose the world .db file for download")
	cmd.Flags().BoolVar(&noFWL, "no-fwl", false, "Do not expose the world .fwl file information")
	cmd.Flags().BoolVar(&noValheim, "no-valheim", false, "Do not run Valheim")

	cmd.Flags().StringVar(&valheimMapWorldVersion, "valheim-map-world-version", "0.221.4", "Version of valheim-map.world to redirect to")

	cmd.Flags().IntVar(&addr, "addr", 8080, "Port for valheimw to listen on")

	cmd.Flags().StringVar(&opts.SaveDir, "savedir", filepath.Join(cache.Dir, "valheim"), "Valheim server -savedir")
	cmd.Flags().StringVar(&opts.Name, "name", "valheimw", "Valheim server -name")
	cmd.Flags().Int64Var(&opts.Port, "port", 0, "Valheim server -port (0 to use default)")
	cmd.Flags().StringVar(&opts.World, "world", "valheimw", "Valheim server -world")
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
