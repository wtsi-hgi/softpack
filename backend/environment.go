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

	reqs, err := s.checkRequiredRecipes(ctx, *env)
	if err != nil {
		return err
	}

	if err := s.db.CreateEnvironment(ctx, *env); err != nil {
		return err
	}

	if len(reqs) > 0 {
		for _, r := range reqs {
			s.waitingEnvs.Append(r, *env) // TODO: Do i need to let the frontend know its waiting?
		}
	} else {
		// build env
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func (s *Server) checkRequiredRecipes(ctx context.Context, env db.Environment) ([]db.RecipeRequest, error) {
	reqs, err := s.db.GetRequestedRecipes(ctx)
	if err != nil {
		return nil, err
	}

	var waitingFor []db.RecipeRequest

	for _, pkg := range env.Packages {
		for _, req := range reqs {
			if db.CheckPkgEqual(pkg, req) {
				waitingFor = append(waitingFor, req)
			}
		}
	}

	return waitingFor, nil
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

func (s *Server) UpdateEnvironment(w http.ResponseWriter, r *http.Request) error {
	u, err := GetItemFromRequest[db.UpdateEnv](r)
	if err != nil {
		return err
	}

	if err := s.db.UpdateEnvironment(r.Context(), *u); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}

// Implementing these below to match api to old frontend, realistically would want to
// swap to only using the general case UpdateEnvironment function.

func (s *Server) AddEnvironmentTag(w http.ResponseWriter, r *http.Request) error {
	env, u, err := s.getEnvFromUpdateIdx(r)
	if err != nil {
		return err
	}

	for _, tag := range env.Tags {
		if tag.Name == u.Value {
			return ErrDuplicateItem
		}
	}

	env.Tags = append(env.Tags, db.Tag{Name: u.Value})

	if err := s.db.UpdateEnvironment(r.Context(), db.UpdateEnv{
		EnvironmentIndex: u.EnvironmentIndex,
		Tags:             &env.Tags,
	}); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func (s *Server) DeleteEnvironmentTag(w http.ResponseWriter, r *http.Request) error {
	env, u, err := s.getEnvFromUpdateIdx(r)
	if err != nil {
		return err
	}

	i := slices.IndexFunc(env.Tags, func(t db.Tag) bool {
		return t.Name == u.Name
	})
	if i < 0 {
		return db.ErrNoRowsAffected
	}

	env.Tags = slices.Delete(env.Tags, i, i+1)

	if err := s.db.UpdateEnvironment(r.Context(), *&db.UpdateEnv{
		EnvironmentIndex: u.EnvironmentIndex,
		Tags:             &env.Tags,
	}); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func (s *Server) getEnvFromUpdateIdx(r *http.Request) (*db.Environment, *db.UpdateValue, error) {
	u, err := GetItemFromRequest[db.UpdateValue](r)
	if err != nil {
		return nil, nil, err
	}

	var env db.Environment
	if err := s.db.WithContext(r.Context()).Preload("Tags").First(&env, u.EnvironmentIndex).Error; err != nil {
		return nil, nil, err
	}

	return &env, u, nil
}

func (s *Server) GetTags(w http.ResponseWriter, r *http.Request) error {
	tags, err := s.db.GetTags(r.Context())
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(tags); err != nil {
		return err
	}

	return nil
}
