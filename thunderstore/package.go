package thunderstore

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	xslice "github.com/frantjc/x/slice"
)

type Package struct {
	Namespace      string    `json:"namespace"`
	Name           string    `json:"name"`
	VersionNumber  string    `json:"version_number,omitempty"`
	FullName       string    `json:"full_name"`
	Owner          string    `json:"owner,omitempty"`
	PackageURL     *URL      `json:"package_url,omitempty"`
	Description    string    `json:"description,omitempty"`
	Icon           *URL      `json:"icon,omitempty"`
	Dependencies   []string  `json:"dependencies,omitempty"`
	DownloadURL    *URL      `json:"download_url,omitempty"`
	Downloads      int       `json:"downloads,omitempty"`
	DateCreated    time.Time `json:"date_created,omitempty"`
	DateUpdated    time.Time `json:"date_updated,omitempty"`
	WebsiteURL     *URL      `json:"website_url,omitempty"`
	RatingScore    int       `json:"rating_score,omitempty"`
	TotalDownloads int       `json:"total_downloads,omitempty"`
	IsActive       bool      `json:"is_active,omitempty"`
	IsPinned       bool      `json:"is_pinned,omitempty"`
	IsDeprecated   bool      `json:"is_deprecated,omitempty"`
	Latest         *Latest   `json:"latest,omitempty"`
	Detail         string    `json:"detail,omitempty"`
}

type Latest struct {
	Namespace     string    `json:"namespace"`
	Name          string    `json:"name"`
	VersionNumber string    `json:"version_number,omitempty"`
	FullName      string    `json:"full_name"`
	Description   string    `json:"description,omitempty"`
	Icon          *URL      `json:"icon,omitempty"`
	Dependencies  []string  `json:"dependencies,omitempty"`
	DownloadURL   *URL      `json:"download_url,omitempty"`
	Downloads     int       `json:"downloads,omitempty"`
	DateCreated   time.Time `json:"date_created,omitempty"`
	WebsiteURL    *URL      `json:"website_url,omitempty"`
	IsActive      bool      `json:"is_active,omitempty"`
}

type URL struct {
	*url.URL
}

func (u *URL) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	// TODO: Properly unescape a JSON string.
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return errors.New("input is not a JSON string")
	}

	v, err := url.Parse(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}

	u.URL = v

	return nil
}

func (u *URL) MarshalJSON() ([]byte, error) {
	return []byte(`"` + u.String() + `"`), nil
}

func (p *Package) String() string {
	if p.VersionNumber != "" {
		return strings.Join([]string{p.Namespace, p.Name, p.VersionNumber}, "-")
	} else if p.Latest != nil {
		return strings.Join([]string{p.Latest.Namespace, p.Latest.Name, p.Latest.VersionNumber}, "-")
	} else if p.FullName != "" {
		return p.FullName
	}

	return strings.Join([]string{p.Namespace, p.Name}, "-")
}

func (p *Package) Versionless() string {
	return strings.Join([]string{p.Namespace, p.Name}, "-")
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
			Namespace:     parts[0],
			Name:          parts[1],
			VersionNumber: parts[2],
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
			Namespace:     parts[0],
			Name:          parts[1],
			VersionNumber: parts[2],
		}, nil
	}

	return nil, fmt.Errorf("unable to parse package %s", s)
}
