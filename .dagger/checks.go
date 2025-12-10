package main

import (
	"context"
	"fmt"
)

// +check
func (m *ValheimwDev) IsFmted(ctx context.Context) error {
	if empty, err := m.Fmt(ctx).IsEmpty(ctx); err != nil {
		return err
	} else if !empty {
		return fmt.Errorf("source is not formatted (run `dagger call fmt`)")
	}

	return nil
}
