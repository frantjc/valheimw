package api

import (
	"net/http"

	"github.com/frantjc/sindri/steamapp/postgres"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type Handler struct {
	Path     string
	Database *postgres.Database
}

func NewHandler(basePath string, db *postgres.Database) http.Handler {
	var (
		h = &Handler{Path: basePath, Database: db}
		r = chi.NewRouter()
	)

	r.Use(middleware.RealIP)
	r.Use(middleware.DefaultLogger)

	r.Route(h.Path, func(r chi.Router) {
		r.Put("/steamapps/{appID}", h.UpsertSteamApp)
		r.Get("/steamapps/{appID}", h.GetSteamApp)
	})

	r.NotFound(http.NotFound)

	return r
}
