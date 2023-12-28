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
	"github.com/frantjc/sindri/internal/clienthelper"
	"github.com/frantjc/sindri/thunderstore"
	"github.com/frantjc/sindri/valheim"
	xtar "github.com/frantjc/x/archive/tar"
	"github.com/mmatczuk/anyflag"
	"github.com/spf13/cobra"
)

// NewSindri is the entrypoint for `sindri`.
func NewSindri() *cobra.Command {
	var (
		addr                 string
		noDownload, modsOnly bool
		beta, betaPassword   string
		mods, rmMods         []string
		root, state          string
		verbosity            int
		playerLists          = &valheim.PlayerLists{}
		opts                 = &valheim.Opts{
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

				root, err := filepath.Abs(root)
				if err != nil {
					return err
				}

				state, err := filepath.Abs(state)
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

				if !noDownload {
					if len(mods) > 0 {
						// Mods first because they're going to be smaller most of
						// the time so it makes the whole process a bit faster.
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

				if len(rmMods) > 0 {
					log.Info("removing mods " + strings.Join(rmMods, ", "))

					if err = s.RemoveMods(ctx, rmMods...); err != nil {
						return err
					}
				}

				mods = append(mods, s.BepInEx.Fullname())

				moddedValheimTar, err := s.Extract(mods...)
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

				errC := make(chan error)

				subCmd, err := valheim.NewCommand(ctx, runDir, opts)
				if err != nil {
					return err
				}
				sindri.LogExec(log, subCmd)
				defer os.RemoveAll(runDir)

				go func() {
					log.Info("running Valheim")

					if err := subCmd.Run(); err != nil {
						errC <- fmt.Errorf("valheim: %w", err)
					} else {
						errC <- nil
					}
				}()

				var (
					seedJSONHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						seed, err := valheim.ReadSeed(opts)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						w.Header().Add("Content-Type", "application/json")

						_, _ = w.Write([]byte(`{"seed":"` + seed + `"}`))
					})
					seedTxtHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						seed, err := valheim.ReadSeed(opts)
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

				if len(mods) > 0 {
					var (
						modTarHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							rc, err := s.ExtractMods(mods...)
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
							rc, err := s.ExtractMods(mods...)
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

					errC <- fmt.Errorf("sindri: %s", srv.Serve(l))
				}()
				defer srv.Close()

				return <-errC
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")
	cmd.Flags().CountVarP(&verbosity, "verbose", "V", "verbosity for sindri")

	cmd.Flags().StringVarP(&root, "root", "r", filepath.Join(xdg.DataHome, "sindri"), "root directory for sindri (-savedir resides here)")
	_ = cmd.MarkFlagDirname("root")

	cmd.Flags().StringVarP(&state, "state", "s", filepath.Join(xdg.RuntimeDir, "sindri"), "state directory for sindri")
	_ = cmd.MarkFlagDirname("state")

	cmd.Flags().StringArrayVarP(&mods, "mod", "m", nil, "Thunderstore mods (case-sensitive)")
	cmd.Flags().StringArrayVar(&rmMods, "rm", nil, "Thunderstore mods to remove (case-sensitive)")
	cmd.Flags().BoolVar(&modsOnly, "mods-only", false, "do not redownload Valheim")
	cmd.Flags().BoolVar(&noDownload, "airgap", false, "do not redownload Valheim or mods")
	cmd.Flags().BoolVar(&noDownload, "no-download", false, "do not redownload Valheim or mods")
	_ = cmd.Flags().MarkHidden("airgap")
	_ = cmd.Flags().MarkDeprecated("airgap", "please use --no-download instead of --airgap")

	cmd.Flags().StringVar(&addr, "addr", ":8080", "address for sindri")

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

	cmd.Flags().StringVar(&beta, "beta", "", "Steam beta branch")
	cmd.Flags().StringVar(&betaPassword, "beta-password", "", "Steam beta password")

	return cmd
}
