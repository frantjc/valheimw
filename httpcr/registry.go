package httpcr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/frantjc/go-ingress"
	"github.com/frantjc/sindri/internal/imgutil"
	xhttp "github.com/frantjc/x/net/http"
	"github.com/go-logr/logr"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/uuid"
)

const (
	HeaderDockerContentDigest = "Docker-Content-Digest"
)

type (
	Manifest = v1.Manifest
	Blob     = v1.Layer
)

type Registry interface {
	HeadManifest(ctx context.Context, name string, reference string) error
	GetManifest(ctx context.Context, name string, reference string) (*Manifest, error)
	HeadBlob(ctx context.Context, name string, digest string) error
	GetBlob(ctx context.Context, name string, digest string) (Blob, error)
}

func Handler(reg Registry) http.Handler {
	return ingress.New(
		ingress.ExactPath("/v2/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reg != nil {
				// OCI does not require this, but the Docker v2 spec include it, and GCR sets this.
				// Docker distribution v2 clients may fallback to an older version if this is not set.
				w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
				w.WriteHeader(http.StatusOK)
				return
			}

			http.NotFound(w, r)
		}), ingress.WithMatchIgnoreSlash),
		ingress.PrefixPath("/v2",
			xhttp.AllowHandler(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if reg == nil {
						http.NotFound(w, r)
						return
					}

					var (
						split    = strings.Split(r.URL.Path, "/")
						lenSplit = len(split)
					)

					if len(split) < 5 {
						http.NotFound(w, r)
						return
					}

					var (
						ep        = split[lenSplit-2]
						name      = strings.Join(split[2:lenSplit-2], "/")
						reference = split[lenSplit-1]
						log       = logr.FromContextOrDiscard(r.Context()).WithValues(
							"method", r.Method,
							"name", name,
							"reference", reference,
							"id", uuid.NewString(),
						)
					)

					log.Info(ep)

					switch ep {
					case "manifests":
						if r.Method == http.MethodHead {
							if err := reg.HeadManifest(r.Context(), name, reference); err != nil {
								log.Error(err, ep)
								http.Error(w, err.Error(), http.StatusInternalServerError)
								return
							}

							w.WriteHeader(http.StatusOK)
							return
						}

						manifest, err := reg.GetManifest(r.Context(), name, reference)
						if err != nil {
							log.Error(err, ep)
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						digest, err := imgutil.GetManifestDigest(manifest)
						if err != nil {
							log.Error(err, ep)
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						w.Header().Set("Content-Type", string(manifest.MediaType))
						w.Header().Set(HeaderDockerContentDigest, digest.String())
						_ = json.NewEncoder(w).Encode(manifest)
						return
					case "blobs":
						if r.Method == http.MethodHead {
							if err := reg.HeadBlob(r.Context(), name, reference); err != nil {
								log.Error(err, ep)
								http.Error(w, err.Error(), http.StatusInternalServerError)
								return
							}

							w.WriteHeader(http.StatusOK)
							return
						}

						blob, err := reg.GetBlob(r.Context(), name, reference)
						if err != nil {
							log.Error(err, ep)
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						hash, err := blob.Digest()
						if err != nil {
							log.Error(err, "blob digest")
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						w.Header().Set(HeaderDockerContentDigest, hash.String())

						rc, err := blob.Compressed()
						if err != nil {
							log.Error(err, "compressed blob reader")
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
						defer rc.Close()

						_, _ = io.Copy(w, rc)
						return
					default:
						http.NotFound(w, r)
						return
					}
				}),
				[]string{http.MethodGet, http.MethodHead},
			),
		),
	)
}
