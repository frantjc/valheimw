package steamworkshopitem

import (
	"context"
	"fmt"

	"github.com/frantjc/go-steamcmd"
)

func CacheKey(_ context.Context, appID, publishedFileID int, opts ...Opt) (string, error) {
	o := &Opts{
		platformType: steamcmd.DefaultPlatformType,
	}

	for _, opt := range opts {
		opt(o)
	}

	return fmt.Sprintf("%d-%d-%s", appID, publishedFileID, o.platformType), nil
}
