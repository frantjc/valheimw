package scanners

import (
	"bytes"
	"context"

	"github.com/frantjc/sindri/internal/stoker/stokercr"
)

type OSV struct{}

func (s OSV) Scan(ctx context.Context, b bytes.Buffer) ([]stokercr.Vuln, error) {
	return nil, nil
}
