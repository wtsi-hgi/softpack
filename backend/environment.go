package backend

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/wtsi-hgi/softpack/db"
)

var (
	ErrInvalidJson = Error{
		err:  errors.New("invalid json"),
		code: http.StatusBadRequest,
	}
	ErrDuplicateItem = Error{
		err:  errors.New("item to add already exists"),
		code: http.StatusBadRequest,
	}
	ErrMissingItem = Error{
		err:  errors.New("item to delete does not exist"),
		code: http.StatusBadRequest,
	}
)

// TODO: I dont like this error handling

func ErrDbFail(err error) Error {
	return Error{
		err:  errors.New(err.Error()),
		code: http.StatusBadRequest,
	}
}

func (s *Server) CreateEnvironment(w http.ResponseWriter, r *http.Request) {
	env, err := GetItemFromRequest[db.Environment](r)
	if err != nil {
		HttpError(w, ErrInvalidJson) // TODO: make this different, i dont like this httperror then return stuff
		return
	}

	s.envMu.Lock()
	defer s.envMu.Unlock()

	if err := s.db.CreateEnvironment(r.Context(), *env); err != nil {
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
	idx, err := GetItemFromRequest[db.EnvironmentIndex](r)
	if err != nil {
		HttpError(w, ErrInvalidJson)
		return
	}

	s.envMu.Lock()
	defer s.envMu.Unlock()

	if err := s.db.DeleteEnvironment(r.Context(), *idx); err != nil {
		HttpError(w, ErrDbFail(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
}

// UpdateEnvironment will update an environment's metadata.
// Given an environment, it will index the database with the environment's path,
// name and version. All other fields of the matching record will be updated to
// match. (hidden status, tags, etc)
func (s *Server) UpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	env, err := GetItemFromRequest[db.Environment](r)
	if err != nil {
		HttpError(w, ErrInvalidJson)
		return
	}

	s.envMu.Lock()
	defer s.envMu.Unlock()

	if err := s.db.UpdateEnvironment(r.Context(), *env); err != nil {
		HttpError(w, ErrDbFail(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
}

// Implementing these below to match api to old frontend, realistically would want to
// swap to only using the general case UpdateEnvironment function.

func (s *Server) AddEnvironmentTag(w http.ResponseWriter, r *http.Request) {
	s.envMu.Lock()
	defer s.envMu.Unlock()

	env, value, err := s.getEnvFromUpdateIdx(r) // TODO: remove updateidx, unnecessary, use env
	if err != nil {
		HttpError(w, ErrDbFail(err))
		return
	}

	if slices.Contains(env.Tags, value) {
		HttpError(w, ErrDuplicateItem)
		return
	}

	env.Tags = append(env.Tags, value)

	if err := s.db.UpdateEnvironment(r.Context(), *env); err != nil {
		HttpError(w, ErrDbFail(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) DeleteEnvironmentTag(w http.ResponseWriter, r *http.Request) {
	s.envMu.Lock()
	defer s.envMu.Unlock()

	env, value, err := s.getEnvFromUpdateIdx(r)
	if err != nil {
		HttpError(w, ErrDbFail(err))
		return
	}

	i := slices.Index(env.Tags, value)
	if i < 0 {
		HttpError(w, ErrMissingItem)
		return
	}

	env.Tags = slices.Delete(env.Tags, i, i+1)

	if err := s.db.UpdateEnvironment(r.Context(), *env); err != nil {
		HttpError(w, ErrDbFail(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) getEnvFromUpdateIdx(r *http.Request) (*db.Environment, string, error) {
	idx, err := GetItemFromRequest[db.UpdateByIndex](r)
	if err != nil {
		return nil, "", err
	}

	var env db.Environment
	if err := s.db.WithContext(r.Context()).First(&env, idx.ToIndex()).Error; err != nil {
		return nil, "", err
	}

	return &env, idx.Value, nil
}

func (s *Server) ToggleEnvironmentHidden(w http.ResponseWriter, r *http.Request) {
	env, err := GetItemFromRequest[db.Environment](r)
	if err != nil {
		HttpError(w, ErrInvalidJson)
		return
	}

	env.Hidden = !env.Hidden

	s.envMu.Lock()
	defer s.envMu.Unlock()

	if err := s.db.UpdateEnvironment(r.Context(), *env); err != nil {
		HttpError(w, ErrDbFail(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
}
