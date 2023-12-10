package xos

import (
	"os"
	"strings"

	"github.com/frantjc/go-fn"
)

func MakePath(s ...string) string {
	return strings.Join(
		fn.Filter(s, func(t string, _ int) bool {
			return t != ""
		}),
		string(os.PathListSeparator),
	)
}
