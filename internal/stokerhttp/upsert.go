package stokerhttp

import (
	"encoding/json"
	"net/http"
	"strconv"

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
		log        = logr.FromContextOrDiscard(r.Context()).WithValues("steamappID", steamappID)
	)
	r = r.WithContext(logr.NewContext(r.Context(), log))

	appID, err := strconv.Atoi(chi.URLParam(r, steamappIDParam))
	if err != nil {
		return newHTTPStatusCodeError(err, http.StatusBadRequest)
	}

	var reqBody SteamappSpec
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		return newHTTPStatusCodeError(err, http.StatusBadRequest)
	}

	row, err := h.Database.UpsertBuildImageOpts(r.Context(), appID, rowFrom(appID, &reqBody))
	if err != nil {
		return err
	}

	metadata, err := getSteamappMetadata(r.Context(), row)
	if err != nil {
		return err
	}

	return respondJSON(w, r, &Steamapp{
		SteamappMetadata: *metadata,
		SteamappSpec:     reqBody,
	})
}

func rowFrom(appID int, d *SteamappSpec) *postgres.BuildImageOptsRow {
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
