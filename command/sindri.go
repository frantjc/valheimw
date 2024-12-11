package command

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/frantjc/go-ingress"
	crname "github.com/google/go-containerregistry/pkg/name"
	crremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
)

func NewSindri() *cobra.Command {
	var (
		addr      string
		verbosity int
		cmd       = &cobra.Command{
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, args []string) error {
				var (
					ctx   = cmd.Context()
					paths = []ingress.Path{
						ingress.ExactPath("/v2/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							// OCI does not require this, but the Docker v2 spec include it, and GCR sets this.
							// Docker distribution v2 clients may fallback to an older version if this is not set.
							w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
							w.WriteHeader(http.StatusOK)
						})),
						ingress.ExactPath("/v2", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
							w.WriteHeader(http.StatusOK)
						})),
						ingress.ExactPath("/v2/_catalog", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							// Don't support the non-standard _catalog API.
							http.Error(w, "_catalog is not supported", http.StatusNotFound)
						})),
						ingress.PrefixPath("/v2", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							var (
								split    = strings.Split(r.URL.Path, "/")
								lenSplit = len(split)
							)

							if len(split) < 5 {
								http.Error(w, "", http.StatusBadRequest)
								return
							}

							var (
								ref  = split[lenSplit-1]
								ep   = split[lenSplit-2]
								name = strings.Join(split[2:lenSplit-2], "/")
							)

							pref, err := crname.ParseReference(fmt.Sprintf("frantjc/sindri/%s:%s", name, ref), crname.WithDefaultRegistry("ghcr.io"))
							if err != nil {
								http.Error(w, err.Error(), http.StatusBadRequest)
								return
							}

							desc, err := crremote.Head(pref, crremote.WithContext(r.Context()), crremote.WithUserAgent(r.UserAgent()))
							if err != nil {
								http.Error(w, err.Error(), http.StatusNotFound)
								return
							}

							fmt.Println(r.Method, ep, name, ref, desc)
							http.Error(w, "not implemented", http.StatusNotFound)
						})),
					}
					srv = &http.Server{
						Addr:              addr,
						ReadHeaderTimeout: time.Second * 5,
						BaseContext: func(_ net.Listener) context.Context {
							return ctx
						},
						Handler: ingress.New(paths...),
					}
				)

				// TODO: I don't think the context cancels this.
				l, err := net.Listen("tcp", addr)
				if err != nil {
					return err
				}
				defer l.Close()

				return srv.Serve(l)
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")
	cmd.Flags().CountVarP(&verbosity, "verbose", "V", "verbosity")

	cmd.Flags().StringVar(&addr, "addr", ":8080", "address")

	return cmd
}
