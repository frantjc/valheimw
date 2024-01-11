package sindri

import (
	"runtime/debug"
	"strings"
)

// VersionCore is the SemVer version core of sindri.
// Meant to be be overridden at build time, but kept
// up-to-date sometimes to best support `go install`.
var VersionCore = "1.3.1"

// SemVer returns the semantic version of forge as built
// from VersionCore and debug build info.
func SemVer() string {
	semver := VersionCore

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		var (
			revision string
			modified bool
		)
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				modified = setting.Value == "true"
			}
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
