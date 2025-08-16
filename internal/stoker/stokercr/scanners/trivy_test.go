// //go:build integration

package scanners

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrivy_Scan(t *testing.T) {
	ctx := context.Background()

	scanner, err := NewTrivy(ctx)
	assert.NoError(t, err)

	vulns, err := scanner.scanFile(ctx, "resources/stoker.tar")
	assert.NoError(t, err)
	assert.Len(t, vulns, 94)
}
