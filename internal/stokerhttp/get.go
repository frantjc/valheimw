package stokerhttp

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-logr/logr"
)

type SteamappList struct {
	Offset    int        `json:"offset"`
	Limit     int        `json:"limit"`
	Steamapps []Steamapp `json:"steamapps"`
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
		return newHTTPStatusCodeError(fmt.Errorf("parse steamapp ID: %w", err), http.StatusBadRequest)
	}

	specRow, err := h.Database.SelectBuildImageOpts(r.Context(), parsedSteamappAppID)
	if err != nil {
		return fmt.Errorf("select build image options: %w", err)
	}

	infoRow, err := h.Database.SelectSteamappInfo(r.Context(), parsedSteamappAppID)
	if err != nil {
		return fmt.Errorf("select steamapp info: %w", err)
	}

	return respondJSON(w, r, &Steamapp{
		SteamappSpec: specFromRow(specRow),
		SteamappInfo: infoFromRow(infoRow),
	})
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

	steamapps := make([]Steamapp, len(rows))
	for i, row := range rows {
		infoRow, err := h.Database.SelectSteamappInfo(r.Context(), row.AppID)
		if err != nil {
			return fmt.Errorf("get steamapp info: %w", err)
		}

		steamapps[i] = Steamapp{
			SteamappSpec: specFromRow(&row),
			SteamappInfo: infoFromRow(infoRow),
		}
	}

	return respondJSON(w, r, &SteamappList{
		Offset:    offset,
		Limit:     limit,
		Steamapps: steamapps,
	})
}
