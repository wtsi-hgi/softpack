package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/wtsi-hgi/softpack/apt"
	"github.com/wtsi-hgi/softpack/db"
)

func (s *Server) CreateEnvironment(w http.ResponseWriter, r *http.Request) error {
	env, err := GetItemFromRequest[db.Environment](r)
	if err != nil {
		return err
	}

	if len(env.Packages) == 0 {
		return db.ErrMissingField
	}

	ctx := r.Context()

	if exists := s.apt.CheckPackagesExist(env.Packages); exists {
		return apt.ErrInvalidPackage
	}

	// TODO: This probably wants to be done in a transaction so the map and db cant become out of sync
	waiting, err := s.checkRequiredRecipes(ctx, *env)
	if err != nil {
		return err
	}

	if !waiting {
		if err := s.db.CreateEnvironment(ctx, *env); err != nil {
			return err
		}

		// build env
	}
	// TODO: Do i need to let the frontend its pending?

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func (s *Server) checkRequiredRecipes(ctx context.Context, env db.Environment) (bool, error) {
	reqs, err := s.db.GetRequestedRecipes(ctx)
	if err != nil {
		return false, err
	}

	shouldWait := false

	for _, pkg := range env.Packages {
		for _, req := range reqs {
			if db.CheckPkgEqual(pkg, req) {
				shouldWait = true

				// s.waitingEnvs[req] = append(s.waitingEnvs[req], &env)
				s.waitingEnvs.Append(req, env)
			}
		}
	}

	return shouldWait, nil
}

func (s *Server) GetEnvironment(w http.ResponseWriter, r *http.Request) error {
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

	if err := s.db.UpdateEnvironment(r.Context(), *env); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}

// Implementing these below to match api to old frontend, realistically would want to
// swap to only using the general case UpdateEnvironment function.

func (s *Server) AddEnvironmentTag(w http.ResponseWriter, r *http.Request) error {
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
	env, value, err := s.getEnvFromUpdateIdx(r)
	if err != nil {
		return err
	}

	i := slices.Index(env.Tags, value)
	if i < 0 {
		return db.ErrNoRowsAffected
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

	if err := s.db.UpdateEnvironment(r.Context(), *env); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}
