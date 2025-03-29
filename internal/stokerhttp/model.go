package stokerhttp

import (
	"time"

	"github.com/frantjc/sindri/steamapp/postgres"
)

type Steamapp struct {
	SteamappSpec
	SteamappInfo
}

type SteamappSpec struct {
	BaseImageRef string    `json:"base_image,omitempty"`
	AptPkgs      []string  `json:"apt_packages,omitempty"`
	LaunchType   string    `json:"launch_type,omitempty"`
	PlatformType string    `json:"platform_type,omitempty"`
	Execs        []string  `json:"execs,omitempty"`
	Entrypoint   []string  `json:"entrypoint,omitempty"`
	Cmd          []string  `json:"cmd,omitempty"`
	DateCreated  time.Time `json:"date_created"`
	DateUpdated  time.Time `json:"date_updated"`
	Locked       bool      `json:"locked"`
}

func specFromRow(row *postgres.BuildImageOptsRow) SteamappSpec {
	return SteamappSpec{
		BaseImageRef: row.BaseImageRef,
		AptPkgs:      row.AptPkgs,
		LaunchType:   row.LaunchType,
		PlatformType: row.PlatformType,
		Execs:        row.Execs,
		Entrypoint:   row.Entrypoint,
		Cmd:          row.Cmd,
		DateCreated:  row.DateCreated,
		DateUpdated:  row.DateUpdated,
		Locked:       row.Locked,
	}
}

type SteamappInfo struct {
	Name    string `json:"name,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

func infoFromRow(row *postgres.SteamappInfoRow) SteamappInfo {
	return SteamappInfo{
		Name:    row.Name,
		IconURL: row.IconURL,
	}
}
