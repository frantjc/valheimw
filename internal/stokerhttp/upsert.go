package stokerhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/frantjc/sindri/internal/appinfoutil"
	"github.com/frantjc/sindri/steamapp/postgres"
	"github.com/go-chi/chi"
	"github.com/go-logr/logr"
	"github.com/lib/pq"
)

// @Summary	Create or update the details of a specific Steamapp ID
// @Accept		application/json
// @Produce	json
// @Param		steamappID	path		int				true	"Steamapp ID"
// @Param		request		body		SteamappSpec	true	"Steamapp detail"
// @Success	200			{object}	Steamapp
// @Failure	400			{object}	Error
// @Failure	415			{object}	Error
// @Failure	500			{object}	Error
// @Router		/steamapps/{steamappID} [post]
// @Router		/steamapps/{steamappID} [put]
func (h *handler) upsertSteamapp(w http.ResponseWriter, r *http.Request) error {
	var (
		steamappID = chi.URLParam(r, steamappIDParam)
		log        = logr.FromContextOrDiscard(r.Context()).WithValues(steamappIDParam, steamappID)
	)
	r = r.WithContext(logr.NewContext(r.Context(), log))

	parsedSteamappAppID, err := strconv.Atoi(chi.URLParam(r, steamappIDParam))
	if err != nil {
		return newHTTPStatusCodeError(err, http.StatusBadRequest)
	}

	var reqBody SteamappSpec
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		return newHTTPStatusCodeError(err, http.StatusBadRequest)
	}

	specRow, err := h.Database.UpsertBuildImageOpts(r.Context(), parsedSteamappAppID, rowFromSpec(parsedSteamappAppID, &reqBody))
	if err != nil {
		return fmt.Errorf("upsert build image options: %w", err)
	}

	info, err := getSteamappInfo(r.Context(), specRow)
	if err != nil {
		return fmt.Errorf("get steam app info: %w", err)
	}

	infoRow, err := h.Database.UpsertSteamappInfo(r.Context(), parsedSteamappAppID, rowFromMeta(info))
	if err != nil {
		return fmt.Errorf("upsert steam app info: %w", err)
	}

	return respondJSON(w, r, &Steamapp{
		SteamappMetadata: metaFromRow(infoRow),
		SteamappSpec: specFromRow(specRow),
	})
}

func getSteamappInfo(ctx context.Context, row *postgres.BuildImageOptsRow) (*SteamappMetadata, error) {
	appInfo, err := appinfoutil.GetAppInfo(ctx, row.AppID)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse("https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps")
	if err != nil {
		return nil, err
	}

	return &SteamappMetadata{
		Name:    appInfo.Common.Name,
		IconURL: u.JoinPath(fmt.Sprint(row.AppID), fmt.Sprintf("%s.jpg", appInfo.Common.Icon)).String(),
	}, nil
}

func rowFromSpec(appID int, d *SteamappSpec) *postgres.BuildImageOptsRow {
	r := &postgres.BuildImageOptsRow{
		AppID:        appID,
		BaseImageRef: d.BaseImageRef,
		AptPkgs:      d.AptPkgs,
		LaunchType:   d.LaunchType,
		PlatformType: d.PlatformType,
		Execs:        d.Execs,
		Entrypoint:   d.Entrypoint,
		Cmd:          d.Cmd,
	}

	if r.AptPkgs == nil {
		r.AptPkgs = pq.StringArray{}
	}

	if r.Execs == nil {
		r.Execs = pq.StringArray{}
	}

	if r.Entrypoint == nil {
		r.Entrypoint = pq.StringArray{}
	}

	if r.Cmd == nil {
		r.Cmd = pq.StringArray{}
	}

	return r
}

func rowFromMeta(d *SteamappMetadata) *postgres.SteamappMetadataRow {
	r := &postgres.SteamappMetadataRow{
		Name:    d.Name,
		IconURL: d.IconURL,
	}

	return r
}
