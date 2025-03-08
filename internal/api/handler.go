package api

import (
	"net/http"
	"path"

	"github.com/frantjc/sindri/steamapp/postgres"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
)

type Opts struct {
	Path     string
	Fallback http.Handler
}

type Opt interface {
	Apply(*Opts)
}

func (o *Opts) Apply(opts *Opts) {
	if o.Fallback != nil {
		if opts != nil {
			if o.Path != "" {
				opts.Path = path.Join("/", o.Path)
			}
			if o.Fallback != nil {
				opts.Fallback = o.Fallback
			}
		}
	}
}

func newOpts(opts ...Opt) *Opts {
	o := &Opts{Fallback: http.NotFoundHandler()}

	for _, opt := range opts {
		opt.Apply(o)
	}

	return o
}

type handler struct {
	Database *postgres.Database
}

func NewHandler(db *postgres.Database, opts ...Opt) http.Handler {
	var (
		o = newOpts(opts...)
		h = &handler{Database: db}
		r = chi.NewRouter()
	)

	r.Use(middleware.RealIP)
	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logr.FromContextOrDiscard(r.Context()).WithValues("request", uuid.NewString())
			log.Info(r.URL.Path, "method", r.Method)
			h.ServeHTTP(w, r.WithContext(logr.NewContext(r.Context(), log)))
		})
	})

	r.Route(path.Join("/", o.Path), func(r chi.Router) {
		r.Put("/steamapps/{appID}", h.UpsertSteamapp)
		r.Get("/steamapps/{appID}", h.GetSteamapp)
	})

	r.NotFound(o.Fallback.ServeHTTP)

	return r
}
