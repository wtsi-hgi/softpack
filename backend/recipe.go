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

type RecipeDescriptionResponse struct {
	Description string `json:"description"`
}

func (s *Server) RequestRecipe(_ http.ResponseWriter, r *http.Request) error {
	req, err := GetItemFromRequest[db.RecipeRequest](r)
	if err != nil {
		return err
	}

	if err = s.db.RequestRecipe(r.Context(), req); err != nil {
		return err
	}

	return nil
}

func (s *Server) GetRequestedRecipes(w http.ResponseWriter, r *http.Request) error {
	reqs, err := s.db.GetRequestedRecipes(r.Context())
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(reqs); err != nil {
		return err
	}

	return nil
}

func (s *Server) GetRecipeDescription(w http.ResponseWriter, r *http.Request) error {
	name, err := GetItemFromRequest[string](r)
	if err != nil {
		return err
	}

	idx := s.getPackageIndex(name)

	desc, err := s.apt.GetRecipeDescription(idx.Name)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	response := RecipeDescriptionResponse{
		Description: desc,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return err
	}

	return nil
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

	if err := json.NewEncoder(w).Encode(pkgs); err != nil {
		return err
	}

	return nil
}

func (s *Server) RemoveRequestedRecipe(_ http.ResponseWriter, r *http.Request) error {
	toDelete, err := GetItemFromRequest[db.RecipeRequest](r)
	if err != nil {
		return err
	}

	if _, exists := s.waitingEnvs.Get(toDelete); exists {
		return ErrEnvUsingRecipe
	}

	if err := s.db.RemoveRequestedRecipe(r.Context(), toDelete); err != nil {
		return err
	}

	return nil
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

	// need to update the packages bit on all envs that were waiting on the recipie with the canonical name and version

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
		s.renameFulfilledPkg(ctx, env, recipie)

		if waiting := s.waitingEnvs.ContainsEnv(env); !waiting {
			s.Build(&env)
		}
	}

	return nil
}

func (s *Server) renameFulfilledPkg(ctx context.Context, env db.Environment, req db.FulfilRequestBody) error {
	if err := s.db.UpdateEnvPackage(ctx, db.UpdateValue[db.FulfilRequestBody]{env.ToIndex(), req}); err != nil {
		return err
	}

	return nil
}
