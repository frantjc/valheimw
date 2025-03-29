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
)

const (
	Scheme = "postgres"
)

type BuildImageOptsRow struct {
	AppID        int            `db:"app_id"`
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

type SteamappMetadataRow struct {
	AppID       int       `db:"app_id"`
	Name        string    `db:"name"`
	IconURL     string    `db:"icon_url"`
	DateCreated time.Time `db:"date_created"`
	DateUpdated time.Time `db:"date_updated"`
	Locked      bool      `db:"locked"`
}

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
			app_id INTEGER PRIMARY KEY,
			date_created TIMESTAMP WITHOUT TIME ZONE NOT NULL,
			date_updated TIMESTAMP WITHOUT TIME ZONE NOT NULL,
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

	q = `
		CREATE TABLE IF NOT EXISTS steamappsinfo (
			app_id INTEGER PRIMARY KEY REFERENCES steamapps(app_id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			icon_url TEXT NOT NULL
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
			app_id,
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
		WHERE app_id = $1;
	`

	var o BuildImageOptsRow
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

func (g *Database) SelectBuildImageOpts(
	ctx context.Context,
	appID int,
) (*BuildImageOptsRow, error) {
	q := `
		SELECT
			app_id,
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
		WHERE app_id = $1;
	`

	var o BuildImageOptsRow
	if err := g.db.GetContext(ctx, &o, q, appID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &o, nil
}

func (g *Database) SelectSteamappInfo(
	ctx context.Context,
	appID int,
) (*SteamappMetadataRow, error) {
	q := `
		SELECT
			sa.app_id,
			sa.date_created,
			sa.date_updated,
			sa.locked,
			sai.name,
			sai.icon_url
		FROM steamapps sa
		INNER JOIN steamappsinfo sai
			ON sa.app_id = sai.app_id
		WHERE sa.app_id = $1;
	`

	var o SteamappMetadataRow
	if err := g.db.GetContext(ctx, &o, q, appID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &o, nil
}

func (g *Database) ListBuildImageOpts(
	ctx context.Context,
	offset, limit int,
) ([]BuildImageOptsRow, error) {
	q := `
		SELECT 
			app_id,
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
		ORDER BY date_updated
		LIMIT $1 OFFSET $2;
	`

	var o []BuildImageOptsRow
	if err := g.db.SelectContext(ctx, &o, q, limit, offset); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []BuildImageOptsRow{}, nil
		}

		return nil, err
	}

	return o, nil
}

func (g *Database) UpsertBuildImageOpts(ctx context.Context, appID int, row *BuildImageOptsRow) (*BuildImageOptsRow, error) {
	q := `
		INSERT INTO steamapps(
			app_id,
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
		ON CONFLICT (app_id)
		DO UPDATE SET
			date_updated = NOW(),
			base_image = $2,
			apt_packages = $3,
			launch_type = $4,
			platform_type = $5,
			execs = $6,
			entrypoint = $7,
			cmd = $8
		RETURNING *;
	`

	var o BuildImageOptsRow
	if err := g.db.QueryRowContext(
		ctx,
		q,
		appID,
		row.BaseImageRef,
		row.AptPkgs,
		row.LaunchType,
		row.PlatformType,
		row.Execs,
		row.Entrypoint,
		row.Cmd,
	).Scan(
		&o.AppID, &o.DateCreated, &o.DateUpdated, &o.BaseImageRef, &o.AptPkgs,
		&o.LaunchType, &o.PlatformType, &o.Execs, &o.Entrypoint, &o.Cmd, &o.Locked,
	); err != nil {
		return nil, err
	}

	return &o, nil
}

func (g *Database) UpsertSteamappInfo(ctx context.Context, appID int, row *SteamappMetadataRow) (*SteamappMetadataRow, error) {
	q := `
		INSERT INTO steamappsinfo(
			app_id,
			name,
			icon_url
		)
		VALUES($1, $2, $3)
		ON CONFLICT (app_id)
		DO UPDATE SET
			name = $2,
			icon_url = $3
		RETURNING *;
	`

	var o SteamappMetadataRow
	if err := g.db.QueryRowContext(
		ctx,
		q,
		appID,
		row.Name,
		row.IconURL,
	).Scan(
		&o.AppID, &o.Name, &o.IconURL,
	); err != nil {
		return nil, err
	}

	return &o, nil
}

func (g *Database) Close() error {
	return g.db.Close()
}
