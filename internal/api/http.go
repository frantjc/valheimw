package api

import (
	"time"

	"github.com/frantjc/sindri/steamapp/postgres"
)

type BuildImageOptsResponse struct {
	AppID        int       `json:"appID"`
	DateCreated  time.Time `json:"dateCreated"`
	DateUpdated  time.Time `json:"dateUpdated"`
	BaseImageRef string    `json:"baseImage"`
	AptPkgs      []string  `json:"aptPackages"`
	LaunchType   string    `json:"launchType"`
	PlatformType string    `json:"platformType"`
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
