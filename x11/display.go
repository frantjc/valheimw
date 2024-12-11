package x11

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	EnvVarDisplay = "DISPLAY"
)

func ParseDisplay(s string) (*Display, error) {
	var (
		display = &Display{}
		err     error
	)

	hostname, displayDotScreen, found := strings.Cut(s, ":")
	if !found {
		return display, nil
	}
	display.Hostname = hostname

	rawDisplay, rawScreen, _ := strings.Cut(displayDotScreen, ".")
	if rawDisplay != "" {
		if display.Display, err = strconv.Atoi(rawDisplay); err != nil {
			return nil, err
		}
	}

	if rawScreen != "" {
		if display.Screen, err = strconv.Atoi(rawScreen); err != nil {
			return nil, err
		}
	}

	return display, nil
}

type Display struct {
	Hostname string
	Display  int
	Screen   int
}

func (d *Display) String() string {
	dotScreen := ""
	if d.Screen > 0 {
		dotScreen = fmt.Sprintf(".%d", d.Screen)
	}

	return fmt.Sprintf("%s:%d%s", d.Hostname, d.Display, dotScreen)
}
