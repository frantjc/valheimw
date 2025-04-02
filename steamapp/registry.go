package steamapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/frantjc/sindri/contreg"
	"github.com/frantjc/sindri/internal/imgutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"gocloud.dev/blob"
	"golang.org/x/sync/errgroup"
)

type PullRegistry struct {
	Database     Database
	ImageBuilder *ImageBuilder
	Bucket       *blob.Bucket
}

var _ contreg.Puller = &PullRegistry{}

func (r *PullRegistry) HeadManifest(ctx context.Context, name string, reference string) error {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return err
	} else if err := ValidateAppID(appID); err != nil {
		return err
	}

	if err := digest.Digest(reference).Validate(); err == nil {
		rc, err := r.Bucket.NewReader(ctx, filepath.Join(name, "manifests", reference), nil)
		if err != nil {
			return err
		}
		defer rc.Close()

		manifest := &contreg.Manifest{}

		if err = json.NewDecoder(rc).Decode(manifest); err != nil {
			return err
		}
	}

	return nil
}

func (r *PullRegistry) GetManifest(ctx context.Context, name string, reference string) (*contreg.Manifest, error) {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return nil, err
	} else if err := ValidateAppID(appID); err != nil {
		return nil, err
	}

	if reference == "latest" {
		// Special handling for mapping the default image tag to the default Steamapp branch name.
		reference = DefaultBranchName
	} else if err = digest.Digest(reference).Validate(); err == nil {
		// If the reference is a digest instead of a Steamapp branch name, it necessarily
		// must have been generated previously to be retrievable.
		rc, err := r.Bucket.NewReader(ctx, filepath.Join(name, "manifests", reference), nil)
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		manifest := &contreg.Manifest{}

		if err = json.NewDecoder(rc).Decode(manifest); err != nil {
			return nil, err
		}

		return manifest, nil
	}

	// At this point, the caller must be asking for a Steamapp branch name, so
	// we have to build it. Check the database to see if we have a known special
	// handling for it.
	opts, err := r.Database.GetBuildImageOpts(ctx, appID, reference)
	if err != nil {
		return nil, err
	}

	key := filepath.Join(name, fmt.Sprintf("%s.tar", reference))

	wc, err := r.Bucket.NewWriter(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		_ = r.Bucket.Delete(context.WithoutCancel(ctx), key)
	}()
	defer wc.Close()

	if err := r.ImageBuilder.BuildImage(ctx, appID, opts, &BuildImageOpts{Output: wc, Beta: reference}); err != nil {
		return nil, err
	}

	image, err := tarball.Image(func() (io.ReadCloser, error) {
		return r.Bucket.NewReader(ctx, key, nil)
	}, nil)
	if err != nil {
		return nil, err
	}

	manifest, err := image.Manifest()
	if err != nil {
		return nil, err
	}

	eg, egctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		dig, err := imgutil.GetManifestDigest(manifest)
		if err != nil {
			return err
		}

		key := filepath.Join(name, "manifests", dig.String())

		if ok, err := r.Bucket.Exists(egctx, key); ok {
			return nil
		} else if err != nil {
			return err
		}

		wc, err := r.Bucket.NewWriter(egctx, key, &blob.WriterOptions{
			ContentType: "application/json",
		})
		if err != nil {
			return err
		}
		defer wc.Close()

		if err = json.NewEncoder(wc).Encode(manifest); err != nil {
			return err
		}

		return wc.Close()
	})

	eg.Go(func() error {
		key := filepath.Join(name, "blobs", manifest.Config.Digest.String())

		if ok, err := r.Bucket.Exists(egctx, key); ok {
			return nil
		} else if err != nil {
			return err
		}

		cfgfb, err := image.RawConfigFile()
		if err != nil {
			return err
		}

		wc, err := r.Bucket.NewWriter(egctx, key, &blob.WriterOptions{
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
		return nil, err
	}

	for _, layer := range layers {
		eg.Go(func() error {
			hash, err := layer.Digest()
			if err != nil {
				return err
			}

			key := filepath.Join(name, "blobs", hash.String())

			if ok, err := r.Bucket.Exists(egctx, key); ok {
				return nil
			} else if err != nil {
				return err
			}

			rc, err := layer.Compressed()
			if err != nil {
				return err
			}
			defer rc.Close()

			mediaType, err := layer.MediaType()
			if err != nil {
				return err
			}

			wc, err := r.Bucket.NewWriter(egctx, key, &blob.WriterOptions{
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
		return nil, err
	}

	return manifest, nil
}

func (r *PullRegistry) HeadBlob(ctx context.Context, name string, digest string) error {
	hash, err := v1.NewHash(digest)
	if err != nil {
		return err
	}

	key := filepath.Join(name, "blobs", hash.String())

	if ok, err := r.Bucket.Exists(ctx, key); ok {
		return nil
	} else if err != nil {
		return err
	}

	return fmt.Errorf("layer not found: %s@%s", name, digest)
}

func (r *PullRegistry) GetBlob(ctx context.Context, name string, digest string) (contreg.Blob, error) {
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

	if ok, err := r.Bucket.Exists(ctx, key); ok {
		rc, err := r.Bucket.NewReader(ctx, key, nil)
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		var (
			configFile = &specs.Image{}
			buf        = new(bytes.Buffer)
		)

		if err = json.NewDecoder(io.TeeReader(rc, buf)).Decode(configFile); err == nil {
			// The media type here must match with the ExportEntry from Buildkitd.
			return static.NewLayer(buf.Bytes(), types.DockerConfigJSON), nil
		}

		return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
			return r.Bucket.NewReader(ctx, key, nil)
		})
	} else if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("layer not found: %s@%s", name, digest)
}
