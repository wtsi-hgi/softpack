package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wtsi-hgi/softpack/apt"
	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/utils"
	"vimagination.zapto.org/httpbuffer"
)

var (
	ErrInvalidJson   = errors.New("invalid json")
	ErrDuplicateItem = errors.New("item to add already exists")
)

type Server struct {
	waitingEnvs utils.WaitingEnvs

	db  *db.DB
	apt *apt.Server
}

func (b *Server) Serve() http.Handler {
	var m http.ServeMux

	m.Handle("/create-environment", handler(b.CreateEnvironment))
	m.Handle("/get-environments", handler(b.GetEnvironment))
	m.Handle("/update-environment", handler(b.UpdateEnvironment))
	m.Handle("/delete-environment", handler(b.DeleteEnvironment))
	m.Handle("/add-tag", handler(b.AddEnvironmentTag))
	m.Handle("/delete-tag", handler(b.DeleteEnvironmentTag))
	m.Handle("/set-hidden", handler(b.ToggleEnvironmentHidden))
	m.Handle("/request-recipe", handler(b.RequestRecipe))
	m.Handle("/requested-recipes", handler(b.GetRequestedRecipes))
	m.Handle("/get-recipe-description", handler(b.GetRecipeDescription))
	m.Handle("/package-collection", handler(b.GetAllPackages))
	m.Handle("/remove-requested-recipe", handler(b.RemoveRequestedRecipe))
	// m.Handle("/fulfil-requested-recipe", handler(b.FulfilRequestedRecipe))
	m.Handle("/groups", handler(b.GetGroups))

	return &m

	// todo

	// /upload - upload artefacts (only needed for tooling).
	// /fulfil-requested-recipe - frontend fulfilment of requested recipe.
	// /remove-requested-recipe - frontend request to remove a requested recipe.
	// /build-status - frontend request for average build times (may not be required).
	// /update-module - tooling request to update non-Softpack module.
	// /groups - frontend request to get all groups for a username.
	// /tags
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

var httpErrors = map[error]int{ // TODO: I dont like this
	io.EOF:                http.StatusBadRequest,
	ErrInvalidJson:        http.StatusBadRequest,
	ErrDuplicateItem:      http.StatusBadRequest,
	db.ErrNoRowsAffected:  http.StatusBadRequest,
	apt.ErrInvalidPackage: http.StatusBadRequest,
	db.ErrMissingField:    http.StatusBadRequest,
}

func responseCode(err error) int {
	for e, resp := range httpErrors {
		if errors.Is(err, e) {
			return resp
		}
	}

	if _, ok := errors.AsType[*json.SyntaxError](err); ok {
		return http.StatusBadRequest
	}

	fmt.Println("Given error not found in httpErrors", err)

	return http.StatusInternalServerError
}

func GetItemFromRequest[T any](r *http.Request) (*T, error) {
	var item T

	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		return nil, err
	}

	return &item, nil
}

func New(packagesURL string) *Server {
	database, _ := db.Connect("sqlite3", ":memory:")
	apt, _ := apt.New(packagesURL, time.Minute)

	s := &Server{
		db:  database,
		apt: apt,

		waitingEnvs: utils.New(),
	}

	s.buildWaitingEnvs()

	return s
}

func (s *Server) buildWaitingEnvs() {
	ctx := context.Background()

	reqs, _ := s.db.GetRequestedRecipes(ctx)
	envs, _ := s.db.GetEnvironments(ctx)

	for _, req := range reqs {
		for _, env := range envs {
			for _, pkg := range env.Packages {
				if db.CheckPkgEqual(pkg, req) { // is this adequate ? could pkg name be pkg@version? should split by @ and check fields?
					s.waitingEnvs.Append(req, env)
				}
			}
		}
	}
}
