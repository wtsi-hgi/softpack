package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/wtsi-hgi/softpack/apt"
	"github.com/wtsi-hgi/softpack/db"
)

var ErrEnvUsingRecipe = errors.New("an environment is waiting for build with requested recipe")

func (s *Server) RequestRecipe(_ http.ResponseWriter, r *http.Request) error {
	req, err := GetItemFromRequest[db.RecipeRequest](r)
	if err != nil {
		return err
	}

	return s.db.RequestRecipe(r.Context(), req)
}

func (s *Server) GetRequestedRecipes(w http.ResponseWriter, r *http.Request) error {
	reqs, err := s.db.GetRequestedRecipes(r.Context())
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(reqs)
}

func (s *Server) GetRecipeDescription(w http.ResponseWriter, r *http.Request) error {
	name, err := GetItemFromRequest[string](r)
	if err != nil {
		return err
	}

	var desc string

	if strings.HasPrefix(name, "*") {
		desc, err = s.getRequestedRecipeDesc(r.Context(), name)
	} else {
		idx := s.getPackageIndex(name)
		desc, err = s.apt.GetRecipeDescription(idx.Name)
	}

	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(desc)
}

func (s *Server) getRequestedRecipeDesc(ctx context.Context, name string) (string, error) {
	reqs, err := s.db.GetRequestedRecipes(ctx)
	if err != nil {
		return "", err
	}

	pkgName := strings.TrimPrefix(name, "*")

	for _, req := range reqs {
		if req.Name == pkgName {
			return req.Details, nil
		}
	}

	return "", apt.ErrInvalidPackage
}

func (s *Server) getPackageIndex(name string) (idx db.PackageIndex) {
	parts := strings.Split(name, "@")
	if len(parts) == 2 { //nolint:mnd
		idx.Name = parts[0]
		idx.Version = parts[1]

		return
	}

	idx.Name = name

	return idx
}

func (s *Server) GetAllPackages(w http.ResponseWriter, _ *http.Request) error {
	pkgs := s.apt.GetAllPackages()

	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(pkgs)
}

func (s *Server) RemoveRequestedRecipe(_ http.ResponseWriter, r *http.Request) error {
	toDelete, err := GetItemFromRequest[db.RecipeRequest](r)
	if err != nil {
		return err
	}

	if _, exists := s.waitingEnvs.Get(toDelete); exists {
		return ErrEnvUsingRecipe
	}

	return s.db.RemoveRequestedRecipe(r.Context(), toDelete)
}

func (s *Server) FulfilRequestedRecipe(_ http.ResponseWriter, r *http.Request) error {
	recipe, err := GetItemFromRequest[db.FulfilRequestBody](r)
	if err != nil {
		return err
	}

	if exists := s.apt.CheckPackageExists(db.Package{
		Name:    recipe.CanonicalName,
		Version: recipe.CanonicalVersion,
	}); !exists {
		return fmt.Errorf("%q, %q, %w", recipe.CanonicalName, recipe.CanonicalVersion, apt.ErrInvalidPackage)
	}

	if envs, exists := s.waitingEnvs.Get(recipe.RecipeRequest); exists {
		s.waitingEnvs.Delete(recipe.RecipeRequest)

		if err := s.updateAndBuildEnvs(r.Context(), envs, recipe); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) updateAndBuildEnvs(ctx context.Context, envs []db.Environment, recipie db.FulfilRequestBody) error {
	for _, env := range envs {
		if err := s.db.FulfilEnvPackage(ctx, db.UpdateValue[db.FulfilRequestBody]{
			EnvironmentIndex: env.ToIndex(),
			Value:            recipie,
		}); err != nil {
			return err
		}

		if waiting := s.waitingEnvs.ContainsEnv(env); !waiting {
			s.Build(&env) //nolint:contextcheck
		}
	}

	return nil
}
