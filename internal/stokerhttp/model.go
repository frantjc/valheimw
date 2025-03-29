package stokerhttp

import (
	"time"

	"github.com/frantjc/sindri/steamapp/postgres"
)

type Steamapp struct {
	SteamappSpec
	SteamappMetadata
}

type SteamappSpec struct {
	BaseImageRef string   `json:"base_image,omitempty"`
	AptPkgs      []string `json:"apt_packages,omitempty"`
	LaunchType   string   `json:"launch_type,omitempty"`
	PlatformType string   `json:"platform_type,omitempty"`
	Execs        []string `json:"execs,omitempty"`
	Entrypoint   []string `json:"entrypoint,omitempty"`
	Cmd          []string `json:"cmd,omitempty"`
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
	}
}

type SteamappMetadata struct {
	AppID       int       `json:"app_id,omitempty"`
	Name        string    `json:"name,omitempty"`
	IconURL     string    `json:"icon_url,omitempty"`
	DateCreated time.Time `json:"date_created,omitempty"`
	DateUpdated time.Time `json:"date_updated,omitempty"`
	Locked      bool      `json:"locked,omitempty"`
}

func metaFromRow(row *postgres.SteamappMetadataRow) SteamappMetadata {
	return SteamappMetadata{
		AppID:       row.AppID,
		Name:        row.Name,
		IconURL:     row.IconURL,
		DateCreated: row.DateCreated,
		DateUpdated: row.DateUpdated,
		Locked:      row.Locked,
	}
}
