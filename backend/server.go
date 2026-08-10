package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
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
	buildingEnvs map[db.EnvironmentIndex]*db.Environment
	buildTimes   utils.BuildTimes

	db     *db.DB
	apt    *apt.Server
	config *config.Config
}

func (b *Server) Serve() http.Handler {
	var m http.ServeMux

	m.Handle("/create-environment", handler(b.CreateEnvironment))
	m.Handle("/get-environments", handler(b.GetEnvironment))
	// m.Handle("/update-environment", handler(b.UpdateEnvironment))
	m.Handle("/delete-environment", handler(b.DeleteEnvironment))
	m.Handle("/add-tag", handler(b.AddEnvironmentTag))
	m.Handle("/set-hidden", handler(b.SetEnvironmentHidden))
	m.Handle("/delete-tag", handler(b.DeleteEnvironmentTag))
	m.Handle("/request-recipe", handler(b.RequestRecipe))
	m.Handle("/requested-recipes", handler(b.GetRequestedRecipes))
	m.Handle("/get-recipe-description", handler(b.GetRecipeDescription))
	m.Handle("/package-collection", handler(b.GetAllPackages))
	m.Handle("/remove-requested-recipe", handler(b.RemoveRequestedRecipe))
	m.Handle("/fulfil-requested-recipe", handler(b.FulfilRequestedRecipe))
	m.Handle("/groups", handler(b.GetGroups))
	m.Handle("/tags", handler(b.GetTags))
	m.Handle("/build-status", handler(b.GetAverageBuildTime))

	return &m

	// todo //nolint:godox

	// /upload - upload artefacts (only needed for tooling).
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

func GetItemFromRequest[T any](r *http.Request) (T, error) {
	var item T

	var buf bytes.Buffer

	n, err := io.Copy(&buf, r.Body)
	if err != nil {
		return item, err
	}

	if n == 0 {
		return item, nil
	}

	if err := json.Unmarshal(buf.Bytes(), &item); err != nil {
		return item, err
	}

	return item, nil
}

func New(config *config.Config) *Server {
	apt, err := apt.New(config.AptIndexSrc, 5*time.Minute) //nolint:mnd
	if err != nil {
		slog.Error("Invalid apt index", "index", config.AptIndexSrc)

		return nil
	}

	database, _ := db.Connect(config.Driver, config.DBConn) //nolint:errcheck

	s := &Server{
		db:     database,
		apt:    apt,
		config: config,

		buildTimes:   utils.NewBuildTimes(),
		waitingEnvs:  utils.NewWaitingEnvs(),
		buildingEnvs: make(map[db.EnvironmentIndex]*db.Environment),
	}

	s.populateServerCaches()

	return s
}

func (b *Server) Run() error {
	return http.ListenAndServe(b.config.ListenAddr, b.Serve()) //nolint:gosec
}

func (b *Server) populateServerCaches() { //nolint:gocognit
	ctx := context.Background()

	reqs, _ := b.db.GetRequestedRecipes(ctx) //nolint:errcheck
	envs, _ := b.db.GetEnvironments(ctx)     //nolint:errcheck

	for _, req := range reqs {
		for _, env := range envs {
			switch env.Status { //nolint:exhaustive
			case db.Building:
				b.buildingEnvs[env.ToIndex()] = &env
			case db.Concretised:
				if err := b.buildTimes.AddBuildTime(env.BuildStart, env.BuildEnd); err != nil {
					slog.Error("Failure adding build times for env", "env", env.ToIndex())
				}
			}

			for _, pkg := range env.Packages {
				if db.CheckPkgEqual(pkg, req) {
					b.waitingEnvs.Append(req, env)
				}
			}
		}
	}
}
