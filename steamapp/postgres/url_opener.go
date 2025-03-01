package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/internal/domain/steamapp/model"
	"github.com/frantjc/sindri/steamapp"
	"github.com/jmoiron/sqlx"
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

func (d *DatabaseURLOpener) OpenDatabase(ctx context.Context, u *url.URL) (steamapp.Database, error) {
	if u.Scheme != Scheme {
		return nil, fmt.Errorf("invalid scheme %s, expected %s", u.Scheme, Scheme)
	}

	return NewDatabase(ctx, u)
}

func NewDatabase(ctx context.Context, u *url.URL) (*Database, error) {
	db, err := sqlx.Open(u.Scheme, u.String())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(5)

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	q := `
		CREATE TABLE IF NOT EXISTS steamapps (
			appid INTEGER PRIMARY KEY,
			date_created TIMESTAMP WITHOUT TIME ZONE NOT NULL,
			date_updated TIMESTAMP WITHOUT time ZONE NOT NULL,
			base_image TEXT NOT NULL,
			apt_packages TEXT[] NOT NULL,
			launch_type TEXT NOT NULL,
			platform_type TEXT NOT NULL,
			execs TEXT[] NOT NULL,
			entrypoint TEXT[] NOT NULL,
			cmd TEXT[] NOT NULL,
			locked BOOLEAN NOT NULL
		);
	`
	if _, err = db.ExecContext(ctx, q); err != nil {
		return nil, err
	}

	return &Database{db}, nil
}

type Database struct {
	db *sqlx.DB
}

var _ steamapp.Database = &Database{}

func (g *Database) GetBuildImageOpts(
	ctx context.Context,
	appID int,
	_ string,
) (*steamapp.GettableBuildImageOpts, error) {
	q := `
		SELECT 
			appid, 
			date_created, 
			date_updated, 
			base_image, 
			apt_packages, 
			launch_type, 
			platform_type, 
			execs, 
			entrypoint, 
			cmd, 
			locked
		FROM steamapps
		WHERE appid = $1;
	`
	var o model.BuildImageOptsRow
	if err := g.db.GetContext(ctx, &o, q, appID); err != nil {
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

func (g *Database) UpsertBuildImageOpts(ctx context.Context, appID int, row model.BuildImageOptsRow) (*model.BuildImageOptsRow, error) {
	q := `
		INSERT INTO steamapps(
			appid, 
			date_created, 
			date_updated, 
			base_image, 
			apt_packages, 
			launch_type, 
			platform_type, 
			execs, 
			entrypoint, 
			cmd, 
			locked
		)
		VALUES($1, NOW(), NOW(), $2, $3, $4, $5, $6, $7, $8, false)
		ON CONFLICT (appid)
		DO UPDATE SET 
			date_updated = NOW(), 
			base_image = $2, \
			apt_packages = $3, 
			launch_type = $4, 
			platform_type = $5, 
			execs = $6, 
			entrypoint = $7, 
			cmd = $8
		RETURNING *;
	`

	var o model.BuildImageOptsRow
	if err := g.db.Select(
		&o, q, 
		appID,
		row.BaseImageRef,
		row.AptPkgs,
		row.LaunchType,
		row.PlatformType,
		row.Execs,
		row.Entrypoint,
		row.Cmd,
	); err != nil {
		return nil, err
	}

	return &o, nil
}

func (g *Database) Close() error {
	return g.db.Close()
}
