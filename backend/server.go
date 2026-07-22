package backend

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/wtsi-hgi/softpack/db"
	"vimagination.zapto.org/httpbuffer"
)

type Server struct {
	envMu sync.RWMutex

	db *db.DB
}

type Error struct {
	Code int
	Err  error
}

func (e Error) Error() string {
	return e.Err.Error()
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

	return &m

	// /upload - upload artefacts (only needed for tooling).
	// /request-recipe - frontend request for new package.
	// /requested-recipes - frontend request to list requested recipes.
	// /fulfil-requested-recipe - frontend fulfilment of requested recipe.
	// /remove-requested-recipe - frontend request to remove a requested recipe.
	// /get-recipe-description - frontend request to get package description for a hover popup.
	// /build-status - frontend request for average build times (may not be required).
	// /create-environment frontend request to create a new environment.
	// /get-environments - frontend request to list environments.
	// /delete-environment - tooling request to delete environment.
	// /add-tag - frontend request to add a tag to an environment.
	// /set-hidden - frontend request to toggle hidden status on environment.
	// /upload-module - tooling request to add non-Softpack module.
	// /update-module - tooling request to update non-Softpack module.
	// /package-collection - frontend request to list all available packages and versions.
	// /groups - frontend request to get all groups for a username.

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

var httpErrors = map[error]int{
	// ErrDuplicateName: http.StatusConflict,
	// ErrInvalidName:   http.StatusUnprocessableEntity,
	// ErrNoModule:      http.StatusNotFound,
	// ErrNoType:        http.StatusNotFound,
	// ErrNoPatch:       http.StatusUnprocessableEntity,
	io.EOF: http.StatusBadRequest,
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

	return http.StatusInternalServerError
}

func GetItemFromRequest[T any](r *http.Request) (*T, error) {
	var item T

	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		return nil, err
	}

	return &item, nil
}

func New() *Server {
	database, _ := db.Connect("sqlite3", ":memory:")

	return &Server{
		db: database,
	}
}
