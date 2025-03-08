package stokerhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/frantjc/sindri/steamapp/postgres"
	"github.com/go-chi/chi"
	"github.com/go-logr/logr"
)

func (h *handler) UpsertSteamapp(w http.ResponseWriter, r *http.Request) {
	logger := logr.FromContextOrDiscard(r.Context())

	appID, err := strconv.Atoi(chi.URLParam(r, "appID"))
	if err != nil {
		logger.Error(err, "failed to convert appID to integer", "appID", appID)
		http.Error(w, "URL param 'appID' must be an integer", http.StatusBadRequest)
		return
	}

	var reqBody BuildImageOptsRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		logger.Error(err, "faild to parse request body")
		http.Error(w, "failed to parse rqeuest body", http.StatusBadRequest)
		return
	}

	row, err := h.Database.UpsertBuildImageOpts(r.Context(), appID, RowFrom(appID, &reqBody))
	if err != nil {
		logger.Error(err, "faild to upsert appID")
		http.Error(w, fmt.Sprintf("failed to upsert appID: %d", appID), http.StatusInternalServerError)
		return
	}

	var response bytes.Buffer
	if err := json.NewEncoder(&response).Encode(ResponseFrom(row)); err != nil {
		logger.Error(err, "faild to encode response")
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(response.Bytes())
}

type BuildImageOptsRequest struct {
	BaseImageRef string   `json:"base_image"`
	AptPkgs      []string `json:"apt_packages"`
	LaunchType   string   `json:"launch_type"`
	PlatformType string   `json:"platform_type"`
	Execs        []string `json:"execs"`
	Entrypoint   []string `json:"entrypoint"`
	Cmd          []string `json:"cmd"`
}

func RowFrom(appID int, r *BuildImageOptsRequest) postgres.BuildImageOptsRow {
	return postgres.BuildImageOptsRow{
		AppID:        appID,
		DateCreated:  time.Time{},
		DateUpdated:  time.Time{},
		BaseImageRef: r.BaseImageRef,
		AptPkgs:      r.AptPkgs,
		LaunchType:   r.LaunchType,
		PlatformType: r.PlatformType,
		Execs:        r.Execs,
		Entrypoint:   r.Entrypoint,
		Cmd:          r.Cmd,
		Locked:       false,
	}
}
