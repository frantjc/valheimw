package api

import (
	"time"

	"github.com/frantjc/sindri/steamapp/postgres"
)

type BuildImageOptsResponse struct {
	AppID        int       `json:"appid"`
	DateCreated  time.Time `json:"date_created"`
	DateUpdated  time.Time `json:"date_updated"`
	BaseImageRef string    `json:"base_image"`
	AptPkgs      []string  `json:"apt_packages"`
	LaunchType   string    `json:"launch_type"`
	PlatformType string    `json:"platform_type"`
	Execs        []string  `json:"execs"`
	Entrypoint   []string  `json:"entrypoint"`
	Cmd          []string  `json:"cmd"`
}

func ResponseFrom(r *postgres.BuildImageOptsRow) BuildImageOptsResponse {
	return BuildImageOptsResponse{
		AppID:        r.AppID,
		DateCreated:  r.DateCreated,
		DateUpdated:  r.DateUpdated,
		BaseImageRef: r.BaseImageRef,
		AptPkgs:      r.AptPkgs,
		LaunchType:   r.LaunchType,
		PlatformType: r.PlatformType,
		Execs:        r.Execs,
	}
}
