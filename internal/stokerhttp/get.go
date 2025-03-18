package stokerhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/frantjc/sindri/internal/appinfoutil"
	"github.com/frantjc/sindri/steamapp/postgres"
	"github.com/go-chi/chi"
	"github.com/go-logr/logr"
)

type SteamappMetadata struct {
	AppID       int       `json:"app_id"`
	Name        string    `json:"name,omitempty"`
	IconURL     string    `json:"icon_url,omitempty"`
	DateCreated time.Time `json:"date_created"`
	DateUpdated time.Time `json:"date_updated"`
	Locked      bool      `json:"locked,omitempty"`
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

type SteamappList struct {
	Offset    int                `json:"offset"`
	Limit     int                `json:"limit"`
	Steamapps []SteamappMetadata `json:"steamapps"`
}

type Steamapp struct {
	SteamappMetadata
	SteamappSpec
}

func newHTTPStatusCodeError(err error, httpStatusCode int) error {
	if err == nil {
		return nil
	}

	if 600 >= httpStatusCode || httpStatusCode < 100 {
		httpStatusCode = 500
	}

	return &httpStatusCodeError{
		err:            err,
		httpStatusCode: httpStatusCode,
	}
}

type httpStatusCodeError struct {
	err            error
	httpStatusCode int
}

func (e *httpStatusCodeError) Error() string {
	if e.err == nil {
		return ""
	}

	return e.err.Error()
}

func (e *httpStatusCodeError) Unwrap() error {
	return e.err
}

func httpStatusCode(err error) int {
	hscerr := &httpStatusCodeError{}
	if errors.As(err, &hscerr) {
		return hscerr.httpStatusCode
	}

	return http.StatusInternalServerError
}

const steamappIDParam = "steamappID"

// @Summary	Get the details for a specific Steamapp ID
// @Produce	json
// @Param		steamappID	path		int	true	"Steamapp ID"
// @Success	200			{object}	Steamapp
// @Failure	400			{object}	Error
// @Failure	415			{object}	Error
// @Failure	500			{object}	Error
// @Router		/steamapps/{steamappID} [get]
func (h *handler) getSteamapp(w http.ResponseWriter, r *http.Request) error {
	var (
		steamappID = chi.URLParam(r, steamappIDParam)
		log        = logr.FromContextOrDiscard(r.Context()).WithValues(steamappIDParam, steamappID)
	)
	r = r.WithContext(logr.NewContext(r.Context(), log))

	parsedSteamappAppID, err := strconv.Atoi(steamappID)
	if err != nil {
		return newHTTPStatusCodeError(fmt.Errorf("parse Steamapp ID: %w", err), http.StatusBadRequest)
	}

	row, err := h.Database.SelectBuildImageOpts(r.Context(), parsedSteamappAppID)
	if err != nil {
		return fmt.Errorf("select build image options: %w", err)
	}

	metadata, err := getSteamappMetadata(r.Context(), row)
	if err != nil {
		return fmt.Errorf("get Steamapp metadata: %w", err)
	}

	return respondJSON(w, r, &Steamapp{
		SteamappMetadata: *metadata,
		SteamappSpec: SteamappSpec{
			BaseImageRef: row.BaseImageRef,
			AptPkgs:      row.AptPkgs,
			LaunchType:   row.LaunchType,
			PlatformType: row.PlatformType,
			Execs:        row.Execs,
		},
	})
}

func getSteamappMetadata(ctx context.Context, row *postgres.BuildImageOptsRow) (*SteamappMetadata, error) {
	appInfo, err := appinfoutil.GetAppInfo(ctx, row.AppID)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse("https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps")
	if err != nil {
		return nil, err
	}

	return &SteamappMetadata{
		AppID:       row.AppID,
		Name:        appInfo.Common.Name,
		IconURL:     u.JoinPath(fmt.Sprint(row.AppID), fmt.Sprintf("%s.jpg", appInfo.Common.Icon)).String(),
		DateCreated: row.DateCreated,
		DateUpdated: row.DateUpdated,
		Locked:      row.Locked,
	}, nil
}

// @Summary	List known Steamapps
// @Produce	json
// @Param		offset	query		int	false	"Offset"
// @Param		limit	query		int	false	"Limit"
// @Success	200		{array}		SteamappMetadata
// @Failure	415		{object}	Error
// @Failure	500		{object}	Error
// @Router		/steamapps [get]
func (h *handler) getSteamapps(w http.ResponseWriter, r *http.Request) error {
	var (
		_     = logr.FromContextOrDiscard(r.Context())
		query = r.URL.Query()
	)

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit < 1 {
		limit = 10
	}

	offset, err := strconv.Atoi(query.Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	rows, err := h.Database.ListBuildImageOpts(r.Context(), offset, limit)
	if err != nil {
		return err
	}

	steamapps := make([]SteamappMetadata, len(rows))
	for i, row := range rows {
		metadata, err := getSteamappMetadata(r.Context(), &row)
		if err != nil {
			return fmt.Errorf("get Steamapp metadata: %w", err)
		}

		steamapps[i] = *metadata
	}

	return respondJSON(w, r, &SteamappList{
		Offset:    offset,
		Limit:     limit,
		Steamapps: steamapps,
	})
}
