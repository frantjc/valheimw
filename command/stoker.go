package command

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/frantjc/sindri/internal/logutil"
	"github.com/frantjc/sindri/internal/stoker"
	"github.com/frantjc/sindri/internal/stoker/stokercr"
	"github.com/frantjc/sindri/internal/stoker/stokercr/controller"
	"github.com/frantjc/sindri/steamapp"
	"github.com/go-openapi/spec"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/appdefaults"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func newExecHandlerWithPortFromEnv(ctx context.Context, name string, args ...string) (http.Handler, *exec.Cmd, error) {
	var (
		cmd = exec.CommandContext(ctx, name, args...)
	)

	lis, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return nil, nil, err
	}

	_, port, err := net.SplitHostPort(lis.Addr().String())
	if err != nil {
		return nil, nil, err
	}

	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%s", port))
	if err != nil {
		return nil, nil, err
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%s", port))

	if err = lis.Close(); err != nil {
		return nil, nil, err
	}

	return httputil.NewSingleHostReverseProxy(target), cmd, nil
}

func NewStoker() *cobra.Command {
	var (
		metricsAddr                                      string
		metricsCertPath, metricsCertName, metricsCertKey string
		webhookCertPath, webhookCertName, webhookCertKey string
		enableLeaderElection                             bool
		probeAddr                                        string
		secureWebhook                                    bool
		secureMetrics                                    bool
		enableHTTP2                                      bool
		addr                                             int
		opts                                             = &stoker.APIHandlerOpts{
			Swagger:     true,
			SwaggerOpts: []func(*spec.Swagger){},
		}
		db        = &stokercr.Database{}
		buildkitd string
		mirror    string
		cmd       = &cobra.Command{
			Use: "stoker",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := ctrl.GetConfig()
				if err != nil {
					return err
				}

				var (
					eg, ctx = errgroup.WithContext(cmd.Context())
					log     = logutil.SloggerFrom(ctx)
					tlsOpts []func(*tls.Config)
				)

				if !enableHTTP2 {
					log.Info("disabling HTTP/2")

					tlsOpts = append(tlsOpts, func(c *tls.Config) {
						c.NextProtos = []string{"http/1.1"}
					})
				}

				var (
					metricsCertWatcher *certwatcher.CertWatcher
					webhookCertWatcher *certwatcher.CertWatcher
					webhookTLSOpts     = tlsOpts
				)

				if len(webhookCertPath) > 0 {
					log.Info("creating webhook cert watcher")

					var err error
					webhookCertWatcher, err = certwatcher.New(
						filepath.Join(webhookCertPath, webhookCertName),
						filepath.Join(webhookCertPath, webhookCertKey),
					)
					if err != nil {
						return err
					}

					webhookTLSOpts = append(webhookTLSOpts, func(config *tls.Config) {
						config.GetCertificate = webhookCertWatcher.GetCertificate
					})
				} else if !secureWebhook {
					webhookTLSOpts = append(webhookTLSOpts, func(config *tls.Config) {
						config.GetCertificate = func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
							return nil, nil
						}
					})
				}

				var (
					webhookServer = webhook.NewServer(webhook.Options{
						TLSOpts: webhookTLSOpts,
					})
					metricsServerOptions = metricsserver.Options{
						BindAddress:   metricsAddr,
						SecureServing: secureMetrics,
						TLSOpts:       tlsOpts,
					}
				)

				if secureMetrics {
					log.Info("securing metrics")

					metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
				}

				if len(metricsCertPath) > 0 {
					log.Info("creating metrics cert watcher")

					var err error
					metricsCertWatcher, err = certwatcher.New(
						filepath.Join(metricsCertPath, metricsCertName),
						filepath.Join(metricsCertPath, metricsCertKey),
					)
					if err != nil {
						return err
					}

					metricsServerOptions.TLSOpts = append(metricsServerOptions.TLSOpts, func(config *tls.Config) {
						config.GetCertificate = metricsCertWatcher.GetCertificate
					})
				}

				scheme, err := stokercr.NewScheme()
				if err != nil {
					return err
				}

				mgr, err := ctrl.NewManager(cfg, ctrl.Options{
					Scheme:                        scheme,
					Metrics:                       metricsServerOptions,
					WebhookServer:                 webhookServer,
					HealthProbeBindAddress:        probeAddr,
					LeaderElection:                enableLeaderElection,
					LeaderElectionID:              "ah7dchz8.sindri.frantj.cc",
					LeaderElectionReleaseOnCancel: true,
				})
				if err != nil {
					return err
				}

				if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
					return err
				}

				if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
					return err
				}

				if len(args) > 0 {
					var ex *exec.Cmd
					opts.Fallback, ex, err = newExecHandlerWithPortFromEnv(ctx, args[0], args[1:]...)
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

				if err := db.SetupWithManager(mgr); err != nil {
					return err
				}

				reconciler := &controller.SteamappReconciler{
					ImageBuilder: &steamapp.ImageBuilder{},
				}

				if mirror != "" {
					if reconciler.Mirror, err = url.Parse(mirror); err != nil {
						return err
					}
				}

				reconciler.ImageBuilder.Client, err = client.New(ctx, buildkitd)
				if err != nil {
					return err
				}

				if err := reconciler.SetupWithManager(mgr); err != nil {
					return err
				}

				eg.Go(func() error {
					log.Info("starting manager")

					return mgr.Start(ctx)
				})

				opts.SwaggerOpts = append(opts.SwaggerOpts, func(s *spec.Swagger) {
					s.Info.Version = cmd.Version
				})

				srv := &http.Server{
					ReadHeaderTimeout: time.Second * 5,
					Handler:           stoker.NewAPIHandler(db, opts),
					BaseContext: func(_ net.Listener) context.Context {
						return cmd.Context()
					},
				}

				eg.Go(func() error {
					log.Info("listening...", "addr", l.Addr().String(), "path", opts.Path)

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

	cmd.Flags().StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service")
	cmd.Flags().StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to")
	cmd.Flags().BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager")
	cmd.Flags().BoolVar(&secureWebhook, "webhook-secure", true,
		"If set, the webhook endpoint is served securely via HTTPS. Use --webhook-secure=false to use HTTP instead")
	cmd.Flags().BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead")
	cmd.Flags().StringVar(&webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate")
	cmd.Flags().StringVar(&webhookCertName, "webhook-cert-name", "", "The name of the webhook certificate file")
	cmd.Flags().StringVar(&webhookCertKey, "webhook-cert-key", "", "The name of the webhook key file")
	cmd.Flags().StringVar(&metricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate")
	cmd.Flags().StringVar(&metricsCertName, "metrics-cert-name", "", "The name of the metrics server certificate file")
	cmd.Flags().StringVar(&metricsCertKey, "metrics-cert-key", "", "The name of the metrics server key file")
	cmd.Flags().BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")

	cmd.Flags().IntVarP(&addr, "addr", "a", 5050, "Port for stoker to listen on")
	cmd.Flags().StringVarP(&opts.Path, "path", "p", "", "Base URL path for stoker")
	cmd.Flags().StringVar(&buildkitd, "buildkitd", appdefaults.Address, "BuildKitd URL for stoker")
	cmd.Flags().StringVar(&mirror, "mirror", "", "Container registry mirror URL for boiler")
	cmd.Flags().StringVarP(&db.Namespace, "namespace", "n", stokercr.DefaultNamespace, "Namespace for stoker")

	return cmd
}
