package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/steamapp"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

const (
	Scheme = "postgres"
)

func init() {
	steamapp.RegisterDatabase(
		new(DatabaseURLOpener),
		Scheme,
	)
}

type DatabaseURLOpener struct{}

func (d *DatabaseURLOpener) OpenDatabase(_ context.Context, u *url.URL) (steamapp.Database, error) {
	if u.Scheme != Scheme {
		return nil, fmt.Errorf("invalid scheme %s, expected %s", u.Scheme, Scheme)
	}

	db, err := sqlx.Open(u.Scheme, u.String())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(5)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Database{db}, nil
}

type Database struct {
	db *sqlx.DB
}

var _ steamapp.Database = &Database{}

func (g Database) GetBuildImageOpts(
	_ context.Context,
	appID int,
	_ string,
) (*steamapp.GettableBuildImageOpts, error) {
	type Opts struct {
		AppID        int            `db:"appid"`
		DateCreated  time.Time      `db:"datecreated"`
		BaseImageRef string         `db:"baseimage"`
		AptPkgs      pq.StringArray `db:"aptpackages"`
		LaunchType   string         `db:"launchtype"`
		PlatformType string         `db:"platformtype"`
		Execs        pq.StringArray `db:"execs"`
		Entrypoint   pq.StringArray `db:"entrypoint"`
		Cmd          pq.StringArray `db:"cmd"`
	}

	q := `
		SELECT appid, datecreated, baseimage, aptpackages, launchtype, platformtype, execs, entrypoint, cmd
		FROM steamapps
		WHERE appid = $1
	`
	var o Opts
	if err := g.db.Get(&o, q, appID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Assume it works out of the box.
			return &steamapp.GettableBuildImageOpts{}, nil
		}

		return nil, err
	}

	return &steamapp.GettableBuildImageOpts{
		BaseImageRef: o.BaseImageRef,
		AptPkgs:      o.AptPkgs,
		LaunchType:   o.LaunchType,
		PlatformType: steamcmd.PlatformType(o.PlatformType),
		Execs:        o.Execs,
		Entrypoint:   o.Entrypoint,
		Cmd:          o.Cmd,
	}, nil
}

func (g Database) Close() error {
	return g.db.Close()
}
