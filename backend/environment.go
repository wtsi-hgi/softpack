package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/wtsi-hgi/softpack/apt"
	"github.com/wtsi-hgi/softpack/db"
)

func (s *Server) CreateEnvironment(_ http.ResponseWriter, r *http.Request) error { //nolint:funlen
	env, err := GetItemFromRequest[db.Environment](r)
	if err != nil {
		return err
	}

	if len(env.Packages) == 0 {
		return db.ErrMissingField
	}

	if exists := s.apt.CheckPackagesExist(env.Packages); exists {
		return apt.ErrInvalidPackage
	}

	ctx := r.Context()

	reqs, err := s.checkRequiredRecipes(ctx, env)
	if err != nil {
		return err
	}

	if err := s.db.CreateEnvironment(ctx, &env); err != nil {
		return err
	}

	if len(reqs) == 0 {
		s.Build(&env)
	}

	for _, r := range reqs {
		env.Status = db.Waiting
		s.waitingEnvs.Append(r, env)
	}

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

func (s *Server) DeleteEnvironment(_ http.ResponseWriter, r *http.Request) error {
	idx, err := GetItemFromRequest[db.EnvironmentIndex](r)
	if err != nil {
		return err
	}

	if err := s.db.DeleteEnvironment(r.Context(), idx); err != nil {
		return err
	}

	return nil
}

func (s *Server) AddEnvironmentTag(_ http.ResponseWriter, r *http.Request) error {
	env, u, err := getUpdateValue[string](s, r)
	if err != nil {
		return err
	}

	for _, tag := range env.Tags {
		if tag.Name == u.Value {
			return ErrDuplicateItem
		}
	}

	env.Tags = append(env.Tags, db.Tag{Name: u.Value})

	if err := s.db.AddEnvironmentTag(r.Context(), db.UpdateValue[db.Tag]{
		EnvironmentIndex: u.EnvironmentIndex,
		Value:            db.Tag{Name: u.Value},
	}); err != nil {
		return err
	}

	return nil
}

func (s *Server) DeleteEnvironmentTag(_ http.ResponseWriter, r *http.Request) error {
	env, u, err := getUpdateValue[string](s, r)
	if err != nil {
		return err
	}

	i := slices.IndexFunc(env.Tags, func(t db.Tag) bool {
		return t.Name == u.Value
	})
	if i < 0 {
		return db.ErrNoRowsAffected
	}

	env.Tags = slices.Delete(env.Tags, i, i+1)

	if err := s.db.DeleteEnvironmentTag(r.Context(), db.UpdateValue[db.Tag]{
		EnvironmentIndex: u.EnvironmentIndex,
		Value:            db.Tag{Name: u.Value},
	}); err != nil {
		return err
	}

	return nil
}

func getUpdateValue[T any](s *Server, r *http.Request) (*db.Environment, *db.UpdateValue[T], error) {
	u, err := GetItemFromRequest[db.UpdateValue[T]](r)
	if err != nil {
		return nil, nil, err
	}

	var env db.Environment
	if err := s.db.WithContext(r.Context()).
		Preload("Tags").
		First(&env, u.EnvironmentIndex).Error; err != nil {
		return nil, nil, err
	}

	return &env, &u, nil
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

func (s *Server) SetEnvironmentHidden(_ http.ResponseWriter, r *http.Request) error {
	_, u, err := getUpdateValue[bool](s, r)
	if err != nil {
		return err
	}

	return s.db.UpdateHidden(r.Context(), *u)
}
