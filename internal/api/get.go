package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-logr/logr"
)

func (h *Handler) GetSteamApp(w http.ResponseWriter, r *http.Request) {
	logger := logr.FromContextOrDiscard(r.Context())

	appID, err := strconv.Atoi(chi.URLParam(r, "appID"))
	if err != nil {
		logger.Error(err, "failed to convert appID to integer", "appID", appID)
		http.Error(w, "URL param 'appID' must be an integer", http.StatusBadRequest)
		return
	}

	row, err := h.Database.SelectBuildImageOpts(r.Context(), appID)
	if err != nil {
		logger.Error(err, "faild to get appID")
		http.Error(w, fmt.Sprintf("failed to get appID: %d", appID), http.StatusInternalServerError)
		return
	}

	var response bytes.Buffer
	if err := json.NewEncoder(&response).Encode(ResponseFrom(row)); err != nil {
		logger.Error(err, "faild to encode response")
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response.Bytes())
}
