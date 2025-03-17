package stokerhttp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"

	"github.com/frantjc/sindri/steamapp/postgres"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-logr/logr"
	"github.com/go-openapi/spec"
	"github.com/google/uuid"
	swagger "github.com/swaggo/http-swagger/v2"
	"github.com/timewasted/go-accept-headers"
)

type Opts struct {
	Path     string
	Fallback http.Handler
	Swagger  bool
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
			if o.Swagger {
				opts.Swagger = true
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

var (
	//go:embed swagger.json
	swaggerJSON []byte
)

func NewAPIHandler(db *postgres.Database, opts ...Opt) http.Handler {
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

	p := path.Join("/", o.Path)
	r.Route(p, func(r chi.Router) {
		if o.Swagger {
			r.Get("/", http.RedirectHandler(path.Join(p, "/index.html"), http.StatusMovedPermanently).ServeHTTP)

			s := &spec.Swagger{}

			if err := json.Unmarshal(swaggerJSON, s); err != nil {
				panic(err)
			}

			s.BasePath = p

			r.Get("/doc.json", func(w http.ResponseWriter, r *http.Request) {
				_ = respondJSON(w, r, s)
			})

			r.Get("/*", swagger.Handler())
		}

		r.Post(fmt.Sprintf("/steamapps/{%s}", steamappIDParam), handleErr(h.upsertSteamapp))
		r.Put(fmt.Sprintf("/steamapps/{%s}", steamappIDParam), handleErr(h.upsertSteamapp))
		r.Get(fmt.Sprintf("/steamapps/{%s}", steamappIDParam), handleErr(h.getSteamapp))
		r.Get("/steamapps", handleErr(h.getSteamapps))
	})

	r.NotFound(o.Fallback.ServeHTTP)

	return r
}

type Error struct {
	Message string `json:"error,omitempty"`
}

func handleErr(handler func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			if nErr := negotiate(w, r, "application/json"); nErr != nil {
				http.Error(w, err.Error(), httpStatusCode(err))
				return
			}

			w.WriteHeader(httpStatusCode(err))
			_ = respondJSON(w, r, &Error{Message: err.Error()})
		}
	}
}

func negotiate(w http.ResponseWriter, r *http.Request, contentType string) error {
	if _, err := accept.Negotiate(r.Header.Get("Accept"), contentType); err != nil {
		w.Header().Set("Accept", contentType)
		return newHTTPStatusCodeError(err, http.StatusUnsupportedMediaType)
	}

	w.Header().Set("Vary", "Accept")

	w.Header().Set("Content-Type", contentType)

	return nil
}

func wantsPretty(r *http.Request) bool {
	pretty, _ := strconv.ParseBool(r.URL.Query().Get("pretty"))
	return pretty
}

func respondJSON(w http.ResponseWriter, r *http.Request, a any) error {
	if err := negotiate(w, r, "application/json"); err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	if wantsPretty(r) {
		enc.SetIndent("", "  ")
	}

	return enc.Encode(a)
}
