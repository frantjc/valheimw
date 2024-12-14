package command

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/frantjc/sindri/distrib"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
)

//go:embed image.tar
var imageTar []byte

func NewBoiler() *cobra.Command {
	var (
		addr     string
		registry = &distrib.SteamappPuller{
			Dir:   "/home/boil/steamapp",
			User:  "boil",
			Group: "boil",
		}
		cmd = &cobra.Command{
			Use:           "boiler",
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, _ []string) error {
				srv := &http.Server{
					Addr:              addr,
					ReadHeaderTimeout: time.Second * 5,
					BaseContext: func(_ net.Listener) context.Context {
						return logr.NewContextWithSlogLogger(cmd.Context(), slog.Default())
					},
					Handler: distrib.Handler(registry),
				}

				l, err := net.Listen("tcp", addr)
				if err != nil {
					return err
				}
				defer l.Close()

				registry.Base, err = tarball.Image(func() (io.ReadCloser, error) {
					return io.NopCloser(bytes.NewReader(imageTar)), nil
				}, nil)
				if err != nil {
					return err
				}

				return srv.Serve(l)
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")

	cmd.Flags().StringVar(&addr, "addr", ":8080", "address")

	cmd.Flags().StringVar(&registry.Username, "username", "", "Steam username")
	cmd.Flags().StringVar(&registry.Password, "password", "", "Steam password")

	return cmd
}
