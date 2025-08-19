package steamapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/frantjc/go-ingress"
	"github.com/frantjc/sindri/internal/httputil"
	"github.com/frantjc/sindri/internal/logutil"
	xhttp "github.com/frantjc/x/net/http"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"gocloud.dev/blob"
	"golang.org/x/sync/errgroup"
)

type PullRegistry struct {
	Database     Database
	ImageBuilder *ImageBuilder
	Bucket       *blob.Bucket
}

func (p *PullRegistry) headManifest(ctx context.Context, name string, reference string) error {
	log := logutil.SloggerFrom(ctx)

	appID, err := strconv.Atoi(name)
	if err != nil {
		return err
	} else if err := ValidateAppID(appID); err != nil {
		return err
	}

	if err := digest.Digest(reference).Validate(); err == nil {
		key := filepath.Join(name, "manifests", reference)

		log.Debug("checking bucket for digest reference", "key", key)

		rc, err := p.Bucket.NewReader(ctx, key, nil)
		if err != nil {
			return err
		}
		defer rc.Close()

		manifest := &v1.Manifest{}

		if err = json.NewDecoder(rc).Decode(manifest); err != nil {
			return err
		}
	}

	// At this point, the caller must be asking for a Steamapp branch name, so
	// we have to build it. Check the database to see if we have a known special
	// handling for it.
	_, err = p.Database.GetBuildImageOpts(ctx, appID, reference)
	if err != nil {
		return err
	}

	return nil
}

func (p *PullRegistry) getManifest(ctx context.Context, name string, reference string) ([]byte, digest.Digest, string, error) {
	log := logutil.SloggerFrom(ctx)

	appID, err := strconv.Atoi(name)
	if err != nil {
		return nil, "", "", err
	} else if err := ValidateAppID(appID); err != nil {
		return nil, "", "", err
	}

	if reference == "latest" {
		// Special handling for mapping the default image tag to the default Steamapp branch name.
		reference = DefaultBranchName
	} else if dig := digest.Digest(reference); dig.Validate() == nil {
		// If the reference is a digest instead of a Steamapp branch name, it necessarily
		// must have been generated previously to be retrievable.
		key := filepath.Join(name, "manifests", reference)

		log.Debug("checking bucket for digest reference", "key", key)

		rc, err := p.Bucket.NewReader(ctx, key, nil)
		if err != nil {
			return nil, "", "", err
		}
		defer rc.Close()

		var (
			manifest = &v1.Manifest{}
			buf      = new(bytes.Buffer)
		)

		if err = json.NewDecoder(io.TeeReader(rc, buf)).Decode(manifest); err != nil {
			return nil, "", "", err
		}

		return buf.Bytes(), dig, string(manifest.MediaType), nil
	}

	// At this point, the caller must be asking for a Steamapp branch name, so
	// we have to build it. Check the database to see if we have a known special
	// handling for it.
	o, err := p.Database.GetBuildImageOpts(ctx, appID, reference)
	if err != nil {
		return nil, "", "", err
	}

	opts := []BuildImageOpt{o, &BuildImageOpts{Beta: reference}}

	// Do a quick initial check on just the manifest. If it's cached already,
	// then so are its blobs, and we can just return the manifest now.
	rawManifest, manifest, err := p.ImageBuilder.getImageManifest(ctx, appID, opts...)
	if err != nil {
		return nil, "", "", err
	}

	dig := digest.FromBytes(rawManifest)

	if err = dig.Validate(); err != nil {
		return nil, "", "", err
	}

	if ok, err := p.Bucket.Exists(ctx, filepath.Join(name, "manifests", dig.String())); ok {
		return rawManifest, dig, string(manifest.MediaType), nil
	} else if err != nil {
		return nil, "", "", err
	}

	key := filepath.Join(name, fmt.Sprintf("%s.tar", reference))

	wc, err := p.Bucket.NewWriter(ctx, key, nil)
	if err != nil {
		return nil, "", "", err
	}
	go func() {
		<-ctx.Done()
		_ = p.Bucket.Delete(context.WithoutCancel(ctx), key)
	}()
	defer wc.Close()

	if err := p.ImageBuilder.BuildImage(ctx, appID, wc, opts...); err != nil {
		return nil, "", "", err
	}

	image, err := tarball.Image(func() (io.ReadCloser, error) {
		return p.Bucket.NewReader(ctx, key, nil)
	}, nil)
	if err != nil {
		return nil, "", "", err
	}

	eg, egctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		key := filepath.Join(name, "manifests", dig.String())

		if ok, err := p.Bucket.Exists(egctx, key); ok {
			return nil
		} else if err != nil {
			return err
		}

		log.Debug("cacheing manifest in bucket", "key", key)

		wc, err := p.Bucket.NewWriter(egctx, key, &blob.WriterOptions{
			ContentType: "application/json",
		})
		if err != nil {
			return err
		}
		defer wc.Close()

		if _, err = wc.Write(rawManifest); err != nil {
			return err
		}

		return wc.Close()
	})

	eg.Go(func() error {
		key := filepath.Join(name, "blobs", manifest.Config.Digest.String())

		if ok, err := p.Bucket.Exists(egctx, key); ok {
			return nil
		} else if err != nil {
			return err
		}

		log.Debug("cacheing config blob in bucket", "key", key)

		cfgfb, err := image.RawConfigFile()
		if err != nil {
			return err
		}

		wc, err := p.Bucket.NewWriter(egctx, key, &blob.WriterOptions{
			ContentType: "application/json",
		})
		if err != nil {
			return err
		}
		defer wc.Close()

		if _, err := wc.Write(cfgfb); err != nil {
			return err
		}

		return wc.Close()
	})

	layers, err := image.Layers()
	if err != nil {
		return nil, "", "", err
	}

	for _, layer := range layers {
		eg.Go(func() error {
			hash, err := layer.Digest()
			if err != nil {
				return err
			}

			key := filepath.Join(name, "blobs", hash.String())

			if ok, err := p.Bucket.Exists(egctx, key); ok {
				return nil
			} else if err != nil {
				return err
			}

			log.Debug("cacheing layer blob in bucket", "key", key, "digest", hash.String())

			rc, err := layer.Compressed()
			if err != nil {
				return err
			}
			defer rc.Close()

			mediaType, err := layer.MediaType()
			if err != nil {
				return err
			}

			wc, err := p.Bucket.NewWriter(egctx, key, &blob.WriterOptions{
				ContentType:     string(mediaType),
				ContentEncoding: "gzip",
			})
			if err != nil {
				return err
			}
			defer wc.Close()

			if _, err := io.Copy(wc, rc); err != nil {
				return err
			}

			return errors.Join(rc.Close(), wc.Close())
		})
	}

	if err = eg.Wait(); err != nil {
		return nil, "", "", err
	}

	return rawManifest, dig, string(manifest.MediaType), nil
}

func (p *PullRegistry) headBlob(ctx context.Context, name string, digest string) error {
	log := logutil.SloggerFrom(ctx)

	hash, err := v1.NewHash(digest)
	if err != nil {
		return err
	}

	key := filepath.Join(name, "blobs", hash.String())

	log.Debug("checking bucket for digest reference", "key", key)

	if ok, err := p.Bucket.Exists(ctx, key); ok {
		return nil
	} else if err != nil {
		return err
	}

	return fmt.Errorf("layer not found: %s@%s", name, digest)
}

func (p *PullRegistry) getBlob(ctx context.Context, name string, digest string) (io.ReadCloser, error) {
	log := logutil.SloggerFrom(ctx)

	appID, err := strconv.Atoi(name)
	if err != nil {
		return nil, err
	} else if err := ValidateAppID(appID); err != nil {
		return nil, err
	}

	hash, err := v1.NewHash(digest)
	if err != nil {
		return nil, err
	}

	key := filepath.Join(name, "blobs", hash.String())

	log.Debug("checking bucket for digest reference", "key", key)

	if ok, err := p.Bucket.Exists(ctx, key); ok {
		return p.Bucket.NewReader(ctx, key, nil)
	} else if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("layer not found: %s@%s", name, digest)
}

const (
	headerDockerContentDigest = "Docker-Content-Digest"
)

func (p *PullRegistry) Handler() http.Handler {
	return ingress.New(
		ingress.ExactPath("/v2/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r != nil {
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
					if p == nil {
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
						log       = logutil.SloggerFrom(r.Context()).With(
							"method", r.Method,
							"name", name,
							"reference", reference,
							"request", uuid.NewString(),
						)
					)

					r = r.WithContext(logutil.SloggerInto(r.Context(), log))
					log.Info(ep)

					switch ep {
					case "manifests":
						if r.Method == http.MethodHead {
							if err := p.headManifest(r.Context(), name, reference); err != nil {
								log.Error(ep, "err", err.Error())
								http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
								return
							}

							w.WriteHeader(http.StatusOK)
							return
						}

						rawManifest, dig, mediaType, err := p.getManifest(r.Context(), name, reference)
						if err != nil {
							log.Error(ep, "err", err.Error())
							http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
							return
						}

						w.Header().Set("Content-Length", fmt.Sprint(len(rawManifest)))
						w.Header().Set("Content-Type", mediaType)
						w.Header().Set(headerDockerContentDigest, dig.String())
						_, _ = w.Write(rawManifest)
						return
					case "blobs":
						if r.Method == http.MethodHead {
							if err := p.headBlob(r.Context(), name, reference); err != nil {
								log.Error(ep, "err", err.Error())
								http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
								return
							}

							w.WriteHeader(http.StatusOK)
							return
						}

						blob, err := p.getBlob(r.Context(), name, reference)
						if err != nil {
							log.Error(ep, "err", err.Error())
							http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
							return
						}
						defer blob.Close()

						w.Header().Set(headerDockerContentDigest, reference)

						_, _ = io.Copy(w, blob)
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
