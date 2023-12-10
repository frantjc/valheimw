package valheim

import (
	"fmt"
)

// Opts is a helper struct to build arguments
// to pass to the Valheim executable.
type Opts struct {
	Port     int64
	World    string
	Name     string
	SaveDir  string
	Password string
}

// ToArgs transforms Opts into an array
// of strings to pass to the Valheim Executable.
func (o *Opts) ToArgs() []string {
	args := []string{}

	if o.SaveDir != "" {
		args = append(args, "-savedir", o.SaveDir)
	}

	if o.Port != 0 {
		args = append(args, "-port", fmt.Sprint(o.Port))
	}

	if o.Name != "" {
		args = append(args, "-name", o.Name)
	}

	if o.World != "" {
		args = append(args, "-world", o.World)
	}

	if o.Password != "" {
		args = append(args, "-password", o.Password)
	}

	return args
}
