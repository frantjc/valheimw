package httpcr

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/frantjc/sindri/internal/imgutil"
	"github.com/opencontainers/go-digest"
	"gocloud.dev/blob"
)

type CacheRegistry struct {
	Registry Registry
	Bucket *blob.Bucket
}

func (r *CacheRegistry) HeadManifest(ctx context.Context, name string, reference string) error {
	if ok, err := r.Bucket.Exists(ctx, filepath.Join(name, "manifests", reference)); err != nil {
		return err
	} else if ok {
		return nil
	}

	return r.Registry.HeadManifest(ctx, name, reference)
}

func (r *CacheRegistry) GetManifest(ctx context.Context, name string, reference string) (*Manifest, Digest, error) {
	key := filepath.Join(name, "manifests", reference)

	if ok, err := r.Bucket.Exists(ctx, key); err != nil {
		return nil, "", err
	} else if ok {
		r, err := r.Bucket.NewReader(ctx, key, nil)
		if err != nil {
			return nil, "", err
		}
		defer r.Close()

		manifest := &Manifest{}

		if err = json.NewDecoder(r).Decode(manifest); err != nil {
			return nil, "", err
		}

		dig := digest.Digest(reference)
		
		if err = dig.Validate(); err != nil {
			dig, err = imgutil.GetManifestDigest(manifest)
			if err != nil {
				return nil, "", err
			}
		}

		return manifest, dig, nil
	}

	manifest, dig, err := r.Registry.GetManifest(ctx, name, reference)
	if err != nil {
		return nil, "", err
	}

	w, err := r.Bucket.NewWriter(ctx, key, nil)
	if err != nil {
		return nil, "", err
	}
	defer w.Close()

	json.NewEncoder(w).Encode(manifest)

	return manifest, dig, nil
}

func (r *CacheRegistry) HeadBlob(ctx context.Context, name string, digest string) error {

}

func (r *CacheRegistry) GetBlob(ctx context.Context, name string, digest string) (Blob, error) {

}
