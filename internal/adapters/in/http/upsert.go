package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/frantjc/sindri/internal/domain/steamapp/model"
	"github.com/go-chi/chi"
)

func (h *Handler) UpsertSteamApp(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.Atoi(chi.URLParam(r, "appID"))
	if err != nil {
		http.Error(w, "URL param 'appID' must be an integer", http.StatusBadRequest)
		return
	}

	var reqBody BuildImageOptsRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse rqeuest body", http.StatusBadRequest)
		return
	}

	opts, err := h.Database.UpsertBuildImageOpts(r.Context(), appID, RowFrom(appID, &reqBody))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to upsert appID: %d", appID), http.StatusInternalServerError)
		return
	}

	var response bytes.Buffer
	if err := json.NewEncoder(&response).Encode(ResponseFrom(opts)); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response"), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(response.Bytes())
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

type BuildImageOptsResponse struct {
	AppID        int       `json:"appid"`
	DateCreated  time.Time `json:"date_created"`
	DateUpdated  time.Time `json:"date_updated"`
	BaseImageRef string    `json:"base_image"`
	AptPkgs      []string  `json:"apt_packages"`
	LaunchType   string    `json:"launch_type"`
	PlatformType string    `json:"platform_type"`
	Execs        []string  `json:"execs"`
	Entrypoint   []string  `json:"entrypoint"`
	Cmd          []string  `json:"cmd"`
}

func ResponseFrom(r *model.BuildImageOptsRow) BuildImageOptsResponse {
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

func RowFrom(appID int, r *BuildImageOptsRequest) model.BuildImageOptsRow {
	return model.BuildImageOptsRow{
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
