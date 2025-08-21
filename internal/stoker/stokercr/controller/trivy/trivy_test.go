package trivy_test

import (
	"io"
	"os"
	"testing"

	"github.com/frantjc/sindri/internal/stoker/stokercr/controller/trivy"
	testdata "github.com/frantjc/sindri/testdata/stoker"
	"github.com/stretchr/testify/assert"
)

func TestScanner_Scan(t *testing.T) {
	ctx := t.Context()

	scanner, err := trivy.NewScanner(ctx, &trivy.ScannerOpts{CacheDir: t.TempDir()})
	assert.NoError(t, err)

	f, err := os.CreateTemp(t.TempDir(), "*.tar")
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = f.Close()
	})

	_, err = f.Write(testdata.Alpine)
	assert.NoError(t, err)

	_, err = f.Seek(0, io.SeekStart)
	assert.NoError(t, err)

	vulns, err := scanner.Scan(ctx, f)
	assert.NoError(t, err)
	assert.Equal(t, len(vulns), 0)
}
