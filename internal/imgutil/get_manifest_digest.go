package imgutil

import (
	"bytes"
	"encoding/json"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/opencontainers/go-digest"
)

func GetManifestDigest(manifest *v1.Manifest) (digest.Digest, error) {
	// TODO: image.Digest() does not produce the correct digest,
	// so we do it here "manually" for now.

	buf := new(bytes.Buffer)

	if err := json.NewEncoder(buf).Encode(manifest); err != nil {
		return "", err
	}

	dig := digest.FromBytes(buf.Bytes())

	return dig, dig.Validate()
}
