package stoker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/frantjc/go-steamcmd"
	"github.com/frantjc/sindri/steamapp"
)

const Scheme = "stoker"

func init() {
	steamapp.RegisterDatabase(
		new(databaseURLOpener),
		Scheme, "https", "http",
	)
}

type databaseURLOpener struct{}

func (d *databaseURLOpener) OpenDatabase(_ context.Context, u *url.URL) (steamapp.Database, error) {
	if !slices.Contains([]string{Scheme, "https", "http"}, u.Scheme) {
		return nil, fmt.Errorf("invalid scheme %s, expected %s", u.Scheme, Scheme)
	}

	if u.Scheme == Scheme {
		v := u.JoinPath()
		v.Scheme = "https"

		return &Client{
			HTTPClient: http.DefaultClient,
			URL:        v,
		}, nil
	}

	return &Client{
		HTTPClient: http.DefaultClient,
		URL:        u,
	}, nil
}

type Client struct {
	HTTPClient *http.Client
	URL        *url.URL
}

// GetBuildImageOpts implements steamapp.Database.
func (c *Client) GetBuildImageOpts(ctx context.Context, appID int, branch string) (*steamapp.GettableBuildImageOpts, error) {
	if branch == "" {
		branch = steamapp.DefaultBranchName
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL.JoinPath("/steamapps", fmt.Sprint(appID), branch).String(), nil)
	if err != nil {
		return nil, err
	}

	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	sa := &Steamapp{}

	if err := json.NewDecoder(res.Body).Decode(sa); err != nil {
		return nil, err
	}

	return &steamapp.GettableBuildImageOpts{
		BaseImageRef: sa.BaseImageRef,
		AptPkgs:      sa.AptPkgs,
		BetaPassword: sa.BetaPassword,
		LaunchType:   sa.LaunchType,
		PlatformType: steamcmd.PlatformType(sa.PlatformType),
		Execs:        sa.Execs,
		Entrypoint:   sa.Entrypoint,
		Cmd:          sa.Cmd,
	}, nil
}
