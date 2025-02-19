package main

import (
	"runtime/debug"
	"strings"
)

// GoReleaser sets these.
var (
	version = "3.1.3"
	commit  = ""
	date    = ""
	builtBy = ""
)

// SemVer returns the semantic version of `mist` as
// built from GoReleaser ldflags and debug build info.
func SemVer() string {
	semver := version

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		var (
			revision string
			modified bool
			_        = date
			_        = builtBy
		)
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				modified = setting.Value == "true"
			}
		}

		if revision == "" {
			revision = commit
		}

		if revision != "" {
			i := len(revision)
			if i > 7 {
				i = 7
			}

			if !strings.Contains(semver, revision[:i]) {
				semver += "+" + revision[:i]
			}
		}

		if modified {
			semver += "*"
		}
	}

	return semver
}
