package steamapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/frantjc/sindri/httpcr"
	"github.com/frantjc/sindri/internal/imgutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/opencontainers/go-digest"
	"gocloud.dev/blob"
	"golang.org/x/sync/errgroup"
)

type Registry struct {
	Base               v1.Image
	Dir                string
	Username, Password string
	User, Group        string
	Bucket             *blob.Bucket
}

var _ httpcr.Registry = &Registry{}

func (r *Registry) HeadManifest(ctx context.Context, name string, reference string) error {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return err
	} else if appID <= 0 {
		return fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	if err := digest.Digest(reference).Validate(); err == nil {
		rc, err := r.Bucket.NewReader(ctx, filepath.Join(name, "manifests", reference), nil)
		if err != nil {
			return err
		}
		defer rc.Close()

		manifest := &httpcr.Manifest{}

		if err = json.NewDecoder(rc).Decode(manifest); err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (r *Registry) GetManifest(ctx context.Context, name string, reference string) (*httpcr.Manifest, error) {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return nil, err
	} else if appID <= 0 {
		return nil, fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	if reference == "latest" {
		reference = DefaultBranchName
	} else if err = digest.Digest(reference).Validate(); err == nil {
		rc, err := r.Bucket.NewReader(ctx, filepath.Join(name, "manifests", reference), nil)
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		manifest := &httpcr.Manifest{}

		if err = json.NewDecoder(rc).Decode(manifest); err != nil {
			return nil, err
		}

		return manifest, nil
	}

	image := r.Base

	cfg, err := r.getConfig(ctx, image, appID, reference)
	if err != nil {
		return nil, err
	}

	image, err = mutate.Config(image, *cfg)
	if err != nil {
		return nil, err
	}

	layer, err := r.getLayer(ctx, appID, reference)
	if err != nil {
		return nil, err
	}

	image, err = mutate.AppendLayers(image, layer)
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

		wc, err := r.Bucket.NewWriter(egctx, key, nil)
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

		layer, err := image.LayerByDigest(manifest.Config.Digest)
		if err != nil {
			return err
		}

		rc, err := layer.Compressed()
		if err != nil {
			return err
		}
		defer rc.Close()

		wc, err := r.Bucket.NewWriter(egctx, key, nil)
		if err != nil {
			return err
		}
		defer wc.Close()

		if _, err := io.Copy(wc, rc); err != nil {
			return err
		}

		return errors.Join(rc.Close(), wc.Close())
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

			wc, err := r.Bucket.NewWriter(egctx, key, nil)
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

func (r *Registry) HeadBlob(ctx context.Context, name string, digest string) error {
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

func (r *Registry) GetBlob(ctx context.Context, name string, digest string) (httpcr.Blob, error) {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return nil, err
	} else if appID <= 0 {
		return nil, fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	hash, err := v1.NewHash(digest)
	if err != nil {
		return nil, err
	}

	key := filepath.Join(name, "blobs", hash.String())

	if ok, err := r.Bucket.Exists(ctx, key); ok {
		return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
			return r.Bucket.NewReader(ctx, key, nil)
		})
	} else if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("layer not found: %s@%s", name, digest)
}

func (r *Registry) getLayer(ctx context.Context, appID int, branch string) (httpcr.Blob, error) {
	layer, err := imgutil.ReproducibleBuildLayerInDirFromOpener(
		func() (io.ReadCloser, error) {
			return Open(ctx, appID,
				WithLogin(r.Username, r.Password, ""),
				WithBeta(branch, ""),
			)
		},
		r.Dir,
		r.User,
		r.Group,
	)
	if err != nil {
		return nil, err
	}

	return layer, nil
}

func (r *Registry) getConfig(ctx context.Context, image v1.Image, appID int, branch string) (*v1.Config, error) {
	cfgf, err := image.ConfigFile()
	if err != nil {
		return nil, err
	}

	return NewImageConfig(
		ctx, appID, &cfgf.Config,
		WithLogin(r.Username, r.Password, ""),
		WithInstallDir(r.Dir),
		WithBeta(branch, ""),
		WithLaunchTypes("server", "default"),
	)
}
