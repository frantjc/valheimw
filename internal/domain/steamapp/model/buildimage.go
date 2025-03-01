package model

import (
	"time"

	"github.com/lib/pq"
)

type BuildImageOptsRow struct {
	AppID        int            `db:"appid"`
	DateCreated  time.Time      `db:"date_created"`
	DateUpdated  time.Time      `db:"date_updated"`
	BaseImageRef string         `db:"base_image"`
	AptPkgs      pq.StringArray `db:"apt_packages"`
	LaunchType   string         `db:"launch_type"`
	PlatformType string         `db:"platform_type"`
	Execs        pq.StringArray `db:"execs"`
	Entrypoint   pq.StringArray `db:"entrypoint"`
	Cmd          pq.StringArray `db:"cmd"`
	Locked       bool           `db:"locked"`
}
