package backend

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/wtsi-hgi/softpack/db"
)

var (
	ErrInvalidJson   = errors.New("invalid json")
	ErrDuplicateItem = errors.New("item to add already exists")
	ErrMissingItem   = errors.New("item to delete does not exist")
)

func (s *Server) CreateEnvironment(w http.ResponseWriter, r *http.Request) error {
	env, err := GetItemFromRequest[db.Environment](r)
	if err != nil {
		return err
	}

	s.envMu.Lock()
	defer s.envMu.Unlock()

	if err := s.db.CreateEnvironment(r.Context(), *env); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func (s *Server) GetEnvironment(w http.ResponseWriter, r *http.Request) error {
	s.envMu.RLock()
	defer s.envMu.RUnlock()

	envs, err := s.db.GetEnvironments(r.Context())
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(envs); err != nil {
		return err
	}

	return nil
}

func (s *Server) DeleteEnvironment(w http.ResponseWriter, r *http.Request) error {
	idx, err := GetItemFromRequest[db.EnvironmentIndex](r)
	if err != nil {
		return err
	}

	s.envMu.Lock()
	defer s.envMu.Unlock()

	if err := s.db.DeleteEnvironment(r.Context(), *idx); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}

// UpdateEnvironment will update an environment's metadata.
// Given an environment, it will index the database with the environment's path,
// name and version. All other fields of the matching record will be updated to
// match. (hidden status, tags, etc)
func (s *Server) UpdateEnvironment(w http.ResponseWriter, r *http.Request) error {
	env, err := GetItemFromRequest[db.Environment](r)
	if err != nil {
		return err
	}

	s.envMu.Lock()
	defer s.envMu.Unlock()

	if err := s.db.UpdateEnvironment(r.Context(), *env); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}

// Implementing these below to match api to old frontend, realistically would want to
// swap to only using the general case UpdateEnvironment function.

func (s *Server) AddEnvironmentTag(w http.ResponseWriter, r *http.Request) error {
	s.envMu.Lock()
	defer s.envMu.Unlock()

	env, value, err := s.getEnvFromUpdateIdx(r) // TODO: remove updateidx, unnecessary, use env
	if err != nil {
		return err
	}

	if slices.Contains(env.Tags, value) {
		return ErrDuplicateItem
	}

	env.Tags = append(env.Tags, value)

	if err := s.db.UpdateEnvironment(r.Context(), *env); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func (s *Server) DeleteEnvironmentTag(w http.ResponseWriter, r *http.Request) error {
	s.envMu.Lock()
	defer s.envMu.Unlock()

	env, value, err := s.getEnvFromUpdateIdx(r)
	if err != nil {
		return err
	}

	i := slices.Index(env.Tags, value)
	if i < 0 {
		return ErrMissingItem
	}

	env.Tags = slices.Delete(env.Tags, i, i+1)

	if err := s.db.UpdateEnvironment(r.Context(), *env); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
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

func (s *Server) ToggleEnvironmentHidden(w http.ResponseWriter, r *http.Request) error {
	env, err := GetItemFromRequest[db.Environment](r)
	if err != nil {
		return err
	}

	env.Hidden = !env.Hidden

	s.envMu.Lock()
	defer s.envMu.Unlock()

	if err := s.db.UpdateEnvironment(r.Context(), *env); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}
