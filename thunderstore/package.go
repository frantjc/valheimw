package thunderstore

import (
	"fmt"
	"regexp"
	"strings"

	xslice "github.com/frantjc/x/slice"
)

type Package struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	Version   string `json:"version_number,omitempty"`
}

func (p *Package) Versionless() *Package {
	return &Package{p.Namespace, p.Name, ""}
}

func (p *Package) Fullname() string {
	return strings.Join(
		xslice.Filter([]string{p.Namespace, p.Name, p.Version}, func(s string, _ int) bool {
			return s != ""
		}),
		"-",
	)
}

func (p *Package) String() string {
	return p.Fullname()
}

func ParsePackage(s string) (*Package, error) {
	var (
		parts    = regexp.MustCompile("[/@:]").Split(s, -1)
		lenParts = len(parts)
	)
	switch {
	case xslice.Some(parts, func(part string, _ int) bool {
		return part == ""
	}):
	case lenParts == 2:
		return &Package{
			Namespace: parts[0],
			Name:      parts[1],
		}, nil
	case lenParts == 3:
		return &Package{
			Namespace: parts[0],
			Name:      parts[1],
			Version:   parts[2],
		}, nil
	}

	return ParsePackageFullname(s)
}

func ParsePackageFullname(s string) (*Package, error) {
	var (
		parts    = strings.Split(s, "-")
		lenParts = len(parts)
	)
	switch {
	case xslice.Some(parts, func(part string, _ int) bool {
		return part == ""
	}):
	case lenParts == 2:
		return &Package{
			Namespace: parts[0],
			Name:      parts[1],
		}, nil
	case lenParts == 3:
		return &Package{
			Namespace: parts[0],
			Name:      parts[1],
			Version:   parts[2],
		}, nil
	}

	return nil, fmt.Errorf("unable to parse package %s", s)
}

type PackageMetadata struct {
	Package      `json:",inline"`
	Description  string           `json:"description,omitempty"`
	Dependencies []string         `json:"dependencies,omitempty"`
	Latest       *PackageMetadata `json:"latest,omitempty"`
}
