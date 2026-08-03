package backend

import (
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

	if err = s.db.RequestRecipe(r.Context(), *req); err != nil {
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

func (s *Server) GetAllPackages(w http.ResponseWriter, r *http.Request) error {
	pkgs := s.apt.GetAllPackages()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(pkgs); err != nil {
		return err
	}

	return nil
}

func (s *Server) RemoveRequestedRecipe(w http.ResponseWriter, r *http.Request) error {
	toDelete, err := GetItemFromRequest[db.RecipeRequest](r)
	if err != nil {
		return err
	}

	if _, exists := s.waitingEnvs.Get(*toDelete); exists {
		return ErrEnvUsingRecipe
	}

	if err := s.db.RemoveRequestedRecipe(r.Context(), *toDelete); err != nil {
		return err
	}

	return nil
}

// Frontend expects to pass in:
// method: POST
//
//	body: JSON.stringify({
//	  name: canonicalName,
//	  version: canonicalVersion,
//	  requestedName: name,
//	  requestedVersion: version
//	})
func (s *Server) FulfilRequestedRecipe(w http.ResponseWriter, r *http.Request) error {
	recipe, err := GetItemFromRequest[db.RecipeRequest](r)
	if err != nil {
		return err
	}

	if envs, exists := s.waitingEnvs.Get(*recipe); exists {
		s.waitingEnvs.Delete(*recipe)

		for _, env := range envs {
			if waiting := s.waitingEnvs.ContainsEnv(env); !waiting {
				// TODO: build environment + add to envs db? is it alr there?
				err := s.Build(env)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
