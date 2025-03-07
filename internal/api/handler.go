package api

import (
	"net/http"

	"github.com/frantjc/sindri/steamapp/postgres"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type Handler struct {
	Database *postgres.Database
}

func NewHandler(db *postgres.Database) http.Handler {
	var (
		h = &Handler{Database: db}
		r = chi.NewRouter()
	)

	r.Use(middleware.RealIP)
	r.Use(middleware.DefaultLogger)

	r.Put("/api/steamapps/{appID}", h.UpsertSteamApp)
	r.Get("/api/steamapps/{appID}", h.GetSteamApp)

	r.NotFound(http.NotFound)

	return r
}
