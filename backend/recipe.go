package backend

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/wtsi-hgi/softpack/db"
)

var ErrEnvUsingRecipe = errors.New("an environment is waiting for build with requested recipe")

type RecipeDescriptionResponse struct {
	Description string `json:"description"`
}

func (s *Server) RequestRecipe(w http.ResponseWriter, r *http.Request) error {
	req, err := GetItemFromRequest[db.RecipeRequest](r)
	if err != nil {
		return err
	}

	s.recMu.Lock()
	defer s.recMu.Unlock()

	if err = s.db.RequestRecipe(r.Context(), *req); err != nil {
		return err
	}

	return nil
}

func (s *Server) GetRequestedRecipes(w http.ResponseWriter, r *http.Request) error {
	s.recMu.Lock()
	defer s.recMu.Unlock()

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

// Frontend expects { "description": "Unknown Module Package" } || {"description": <>}
func (s *Server) GetRecipeDescription(w http.ResponseWriter, r *http.Request) error {
	name, err := GetItemFromRequest[string](r)
	if err != nil {
		return err
	}

	idx := s.getPackageIndex(*name)

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

// name is pkg@version
func (s *Server) getPackageIndex(name string) (idx db.PackageIndex) {
	parts := strings.Split(name, "@")
	if len(parts) == 2 {
		idx.Name = parts[0]
		idx.Version = parts[1]

		return
	}

	idx.Name = name

	return idx
}

// it expects []item where item = { name: string; versions: string[]; }
func (s *Server) GetAllPackages(w http.ResponseWriter, r *http.Request) error {
	pkgs := s.apt.GetAllPackages()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(pkgs); err != nil {
		return err
	}

	return nil
}

// Frontend expects to pass in:
// method: "POST",
//
//	body: JSON.stringify({
//	  name,
//	  version
//	})
//
// Frontend expects back:
// { message : "" ; error : "" }
func (s *Server) RemoveRequestedRecipe(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	toDelete, err := GetItemFromRequest[db.RecipeRequest](r)
	if err != nil {
		return err
	}

	s.recMu.Lock()
	defer s.recMu.Unlock()

	if err := s.checkRecipeExists(ctx, *toDelete); err != nil {
		return err
	}

	if err := s.checkNoDependentEnvs(*toDelete); err != nil {
		return err
	}

	if err := s.db.RemoveRequestedRecipe(r.Context(), *toDelete); err != nil {
		return err
	}

	return nil
}

func (s *Server) checkRecipeExists(ctx context.Context, recipe db.RecipeRequest) error {
	reqs, err := s.db.GetRequestedRecipes(ctx)
	if err != nil {
		return err
	}

	for _, req := range reqs {
		if sameRecipe(req, recipe) {
			return nil
		}
	}

	return db.ErrMissingItem
}

func (s *Server) checkNoDependentEnvs(recipe db.RecipeRequest) error {
	for _, requests := range s.waitingEnvs {
		for _, req := range requests {
			if sameRecipe(req, recipe) {
				return ErrEnvUsingRecipe
			}
		}
	}

	return nil
}

func sameRecipe(a, b db.RecipeRequest) bool {
	return a.Name == b.Name && a.Version == b.Version
}

func (s *Server) FulfilRequestedRecipe(w http.ResponseWriter, r *http.Request) error {
	// Frontend expects to pass in:
	// method: POST
	// body: JSON.stringify({
	//   name: canonicalName,
	//   version: canonicalVersion,
	//   requestedName: name,
	//   requestedVersion: version
	// })
	//
	// Frontend expects back:
	// { message : "" ; error : "" }

	return nil
}
