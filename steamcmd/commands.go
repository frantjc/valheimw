package steamcmd

import "strings"

// Commands is a helper struct to build arguments
// to pass to `steamcmd`.
type Commands struct {
	ForceInstallDir string
	Login           string
	AppUpdate       string
	Validate        bool
	Beta            string
	BetaPassword    string
}

// ToArgs transforms Commands into an array
// of strings to pass to `steamcmd`.
func (c *Commands) ToArgs() []string {
	args := []string{}

	if c.ForceInstallDir != "" {
		args = append(args, "+force_install_dir", strings.TrimSpace(c.ForceInstallDir))
	}

	if c.Login != "" {
		args = append(args, "+login", strings.TrimSpace(c.Login))
	} else {
		args = append(args, "+login", "anonymous")
	}

	if c.AppUpdate != "" {
		args = append(args, "+app_update", strings.TrimSpace(c.AppUpdate))
	}

	if c.Beta != "" {
		args = append(args, "-beta", strings.TrimSpace(c.Beta))
	}

	if c.BetaPassword != "" {
		args = append(args, "-betapassword", strings.TrimSpace(c.BetaPassword))
	}

	if c.Validate {
		args = append(args, "-validate")
	}

	return append(args, "+quit")
}
