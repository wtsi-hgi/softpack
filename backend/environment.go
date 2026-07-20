package backend

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/wtsi-hgi/softpack/db"
)

var (
	ErrInvalidJson = Error{
		err:  errors.New("invalid json"),
		code: http.StatusBadRequest,
	}
)

// TODO: I dont like this error handling

func ErrDbFail(err error) Error {
	return Error{
		err:  errors.New(err.Error()),
		code: http.StatusInternalServerError,
	}
}

func (s *Server) CreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var env db.Environment

	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		HttpError(w, ErrInvalidJson) // TODO: make this different, i dont like this httperror then return stuff
		return
	}

	s.envMu.Lock()
	defer s.envMu.Unlock()

	if err := s.db.CreateEnvironment(r.Context(), env); err != nil {
		HttpError(w, ErrDbFail(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) GetEnvironment(w http.ResponseWriter, r *http.Request) {
	s.envMu.RLock()
	defer s.envMu.RUnlock()

	envs, err := s.db.GetEnvironments(r.Context())
	if err != nil {
		HttpError(w, ErrDbFail(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(envs); err != nil {
		HttpError(w, ErrInvalidJson)
	}
}

func (s *Server) DeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	var idx db.EnvironmentIndex

	if err := json.NewDecoder(r.Body).Decode(&idx); err != nil {
		HttpError(w, ErrInvalidJson)
		return
	}

	s.envMu.Lock()
	defer s.envMu.Unlock()

	if err := s.db.DeleteEnvironment(r.Context(), idx); err != nil {
		HttpError(w, ErrDbFail(err))
	}

	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) UpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	s.envMu.Lock()
	defer s.envMu.Unlock()

	var env db.Environment

	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		HttpError(w, ErrInvalidJson)
		return
	}

	// idx := db.EnvironmentIndex{
	// 	Name: env.Name,
	// 	Path: env.Path,
	// 	Version: env.Version,
	// }

	// updates := map[string]interface{}{

	// }

	if err := s.db.UpdateEnvironment(r.Context(), env); err != nil {
		HttpError(w, ErrDbFail(err))
	}

	w.Header().Set("Content-Type", "application/json")
}

// func (s *Server) AddEnvironmentTag(w http.ResponseWriter, r *http.Request)       {}
// func (s *Server) DeleteEnvironmentTag(w http.ResponseWriter, r *http.Request)    {}
// func (s *Server) ToggleEnvironmentHidden(w http.ResponseWriter, r *http.Request) {}
