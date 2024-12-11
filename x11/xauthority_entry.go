package x11

import (
	"fmt"
	"strings"
)

func ParseXAuthorityEntry(line string) (*XAuthorityEntry, error) {
	parts := strings.Split(line, "  ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("parse XAuthority entry: %s", line)
	}

	display, err := ParseDisplay(parts[0])
	if err != nil {
		return nil, err
	}

	return &XAuthorityEntry{
		Display: display,
		Proto:   parts[1],
		Cookie:  parts[2],
	}, nil
}

type XAuthorityEntry struct {
	Display *Display
	Proto   string
	Cookie  string
}

func (x *XAuthorityEntry) String() string {
	proto := x.Proto
	if proto == "" {
		proto = "."
	}

	return strings.Join([]string{x.Display.String(), x.Proto, x.Cookie}, "  ")
}
