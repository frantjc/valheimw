package steamapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/frantjc/go-kv"
	"github.com/frantjc/sindri/httpcr"
	"github.com/frantjc/sindri/internal/imgutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/opencontainers/go-digest"
)

type Registry struct {
	Base               v1.Image
	Dir                string
	Username, Password string
	User, Group        string
	Store              kv.Store
}

var _ httpcr.Registry = &Registry{}

func (r *Registry) HeadManifest(_ context.Context, name string, _ string) error {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return err
	} else if appID <= 0 {
		return fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	return nil
}

var errValNotFound = errors.New("value not found")

func (r *Registry) setManifest(ctx context.Context, appID int, reference string, manifest *httpcr.Manifest) error {
	return r.Store.Set(ctx, fmt.Sprintf("manifest::%d::%s", appID, reference), manifest)
}

func (r *Registry) getManifest(ctx context.Context, appID int, reference string) (*httpcr.Manifest, error) {
	manifest := &httpcr.Manifest{}
	if ok, err := r.Store.Get(ctx, fmt.Sprintf("manifest::%d::%s", appID, reference), manifest); ok {
		return manifest, nil
	} else if err != nil {
		return nil, err
	}

	return nil, errValNotFound
}

func (r *Registry) setBranch(ctx context.Context, appID int, digest, branch string) error {
	return r.Store.Set(ctx, fmt.Sprintf("branch::%d::%s", appID, digest), branch)
}

func (r *Registry) getBranch(ctx context.Context, appID int, digest string) (string, error) {
	branch := ""
	if ok, err := r.Store.Get(ctx, fmt.Sprintf("branch::%d::%s", appID, digest), &branch); ok {
		return branch, nil
	} else if err != nil {
		return "", err
	}

	return "", errValNotFound
}

func (r *Registry) init(ctx context.Context) error {
	if r.Store == nil {
		var err error
		r.Store, err = kv.Open(ctx, "mem://")
		return err
	}
	return nil
}

func (r *Registry) GetManifest(ctx context.Context, name string, reference string) (*httpcr.Manifest, httpcr.Digest, error) {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return nil, "", err
	} else if appID <= 0 {
		return nil, "", fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	if reference == "latest" {
		reference = DefaultBranchName
	}

	if err := r.init(ctx); err != nil {
		return nil, "", err
	}

	if manifest, err := r.getManifest(ctx, appID, reference); err == nil {
		return manifest, "", nil
	} else if !errors.Is(err, errValNotFound) {
		return nil, "", err
	}

	image := r.Base

	cfg, err := r.getConfig(ctx, image, appID, reference)
	if err != nil {
		return nil, "", err
	}

	image, err = mutate.Config(image, *cfg)
	if err != nil {
		return nil, "", err
	}

	layer, err := r.getLayer(ctx, appID, reference)
	if err != nil {
		return nil, "", err
	}

	image, err = mutate.AppendLayers(image, layer)
	if err != nil {
		return nil, "", err
	}

	manifest, err := image.Manifest()
	if err != nil {
		return nil, "", err
	}

	// TODO: image.Digest() does not produce the correct digest,
	// so we do it here "manually" for now.

	buf := new(bytes.Buffer)

	if err = json.NewEncoder(buf).Encode(manifest); err != nil {
		return nil, "", err
	}

	digest := digest.FromBytes(buf.Bytes())

	if err = r.setManifest(ctx, appID, digest.String(), manifest); err != nil {
		return nil, "", err
	}

	if err = r.setBranch(ctx, appID, manifest.Config.Digest.String(), reference); err != nil {
		return nil, "", err
	}

	return manifest, digest, nil
}

func (r *Registry) HeadBlob(_ context.Context, name string, _ string) error {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return err
	} else if appID <= 0 {
		return fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	return nil
}

func (r *Registry) GetBlob(ctx context.Context, name string, digest string) (httpcr.Blob, error) {
	appID, err := strconv.Atoi(name)
	if err != nil {
		return nil, err
	} else if appID <= 0 {
		return nil, fmt.Errorf("invalid Steam app ID: %d", appID)
	}

	image := r.Base

	if err := r.init(ctx); err != nil {
		return nil, err
	}

	if branch, err := r.getBranch(ctx, appID, digest); err == nil {
		layer, err := r.getLayer(ctx, appID, branch)
		if err != nil {
			return nil, err
		}

		image, err = mutate.AppendLayers(image, layer)
		if err != nil {
			return nil, err
		}

		cfg, err := r.getConfig(ctx, image, appID, branch)
		if err != nil {
			return nil, err
		}

		image, err = mutate.Config(image, *cfg)
		if err != nil {
			return nil, err
		}
	} else if !errors.Is(err, errValNotFound) {
		return nil, err
	}

	hash, err := v1.NewHash(digest)
	if err != nil {
		return nil, err
	}

	return image.LayerByDigest(hash)
}

func (r *Registry) getLayer(ctx context.Context, appID int, branch string) (httpcr.Blob, error) {
	layer, err := imgutil.ReproducibleBuildLayerInDirFromOpener(
		func() (io.ReadCloser, error) {
			return Open(ctx, appID,
				WithLogin(r.Username, r.Password, ""),
				WithBeta(branch, ""),
				WithStore(r.Store),
			)
		},
		r.Dir,
		r.User,
		r.Group,
	)
	if err != nil {
		return nil, err
	}

	hash, err := layer.Digest()
	if err != nil {
		return nil, err
	}

	if err = r.setBranch(ctx, appID, hash.String(), branch); err != nil {
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
		WithStore(r.Store),
	)
}
