package trivy_test

// import (
// 	"context"
// 	"testing"

// 	"github.com/frantjc/sindri/internal/stoker/stokercr/controller/trivy"
// 	"github.com/stretchr/testify/assert"
// )

// func TestTrivy_Scan(t *testing.T) {
// 	ctx := context.Background()

// 	scanner, err := trivy.NewTrivy(ctx)
// 	assert.NoError(t, err)

// 	vulns, err := scanner.Scan(ctx, "resources/debian.tar")
// 	assert.NoError(t, err)
// 	assert.Len(t, vulns, 51)
// }
