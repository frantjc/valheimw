package cache

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

var (
	Dir = filepath.Join(xdg.CacheHome, "sindri")
)
