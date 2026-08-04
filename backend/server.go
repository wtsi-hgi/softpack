package backend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/wtsi-hgi/softpack/apt"
	"github.com/wtsi-hgi/softpack/config"
	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/utils"
	"vimagination.zapto.org/httpbuffer"
)

var (
	ErrInvalidJSON   = errors.New("invalid json")
	ErrDuplicateItem = errors.New("item to add already exists")
)

type Server struct {
	waitingEnvs  utils.WaitingEnvs
	buildingEnvs map[uint]BuildingEnv

	db     *db.DB
	apt    *apt.Server
	config *config.Config
}

func (b *Server) Serve() http.Handler {
	var m http.ServeMux

	m.Handle("/create-environment", handler(b.CreateEnvironment))
	m.Handle("/get-environments", handler(b.GetEnvironment))
	m.Handle("/update-environment", handler(b.UpdateEnvironment))
	m.Handle("/delete-environment", handler(b.DeleteEnvironment))
	m.Handle("/add-tag", handler(b.AddEnvironmentTag))
	m.Handle("/delete-tag", handler(b.DeleteEnvironmentTag))
	m.Handle("/request-recipe", handler(b.RequestRecipe))
	m.Handle("/requested-recipes", handler(b.GetRequestedRecipes))
	m.Handle("/get-recipe-description", handler(b.GetRecipeDescription))
	m.Handle("/package-collection", handler(b.GetAllPackages))
	m.Handle("/remove-requested-recipe", handler(b.RemoveRequestedRecipe))
	// m.Handle("/fulfil-requested-recipe", handler(b.FulfilRequestedRecipe))
	m.Handle("/groups", handler(b.GetGroups))
	m.Handle("/tags", handler(b.GetTags))
	m.Handle("/build-status", handler(b.GetAverageBuildTime))

	return &m

	// todo //nolint:godox

	// /upload - upload artefacts (only needed for tooling).
	// /build-status - frontend request for average build times (may not be required).
	// /update-module - tooling request to update non-Softpack module.
}

type handler func(w http.ResponseWriter, r *http.Request) error

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httpbuffer.Handler{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := h(w, r); err != nil {
				http.Error(w, err.Error(), responseCode(err))
			}
		}),
	}.ServeHTTP(w, r)
}

var httpErrors = []error{
	io.EOF,
	ErrInvalidJSON,
	ErrDuplicateItem,
	db.ErrNoRowsAffected,
	apt.ErrInvalidPackage,
	db.ErrMissingField,
}

func responseCode(err error) int {
	if slices.Contains(httpErrors, err) {
		return http.StatusBadRequest
	}

	if _, ok := errors.AsType[*json.SyntaxError](err); ok { //nolint:errcheck
		return http.StatusBadRequest
	}

	log.Printf("Given error not found in httpErrors %s", err)

	return http.StatusInternalServerError
}

func GetItemFromRequest[T any](r *http.Request) (*T, error) {
	var item T

	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		return nil, err
	}

	return &item, nil
}

func New(config *config.Config) *Server {
	apt, _ := apt.New(config.AptSrc, time.Minute)       //nolint:errcheck
	database, _ := db.Connect("sqlite3", config.DBConn) //nolint:errcheck

	s := &Server{
		db:     database,
		apt:    apt,
		config: config,

		waitingEnvs: utils.New(),
	}

	s.generateWaitingEnvs()

	return s
}

func (b *Server) Run() error {
	return http.ListenAndServe(b.config.ListenAddr, b.Serve())
}

func (b *Server) generateWaitingEnvs() {
	ctx := context.Background()

	reqs, _ := b.db.GetRequestedRecipes(ctx) //nolint:errcheck
	envs, _ := b.db.GetEnvironments(ctx)     //nolint:errcheck

	for _, req := range reqs {
		for _, env := range envs {
			for _, pkg := range env.Packages {
				if db.CheckPkgEqual(pkg, req) {
					b.waitingEnvs.Append(req, env)
				}
			}
		}
	}
}

func ptrTo[T any](v T) *T {
	return &v
}
