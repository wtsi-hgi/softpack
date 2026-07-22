package backend

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/wtsi-hgi/softpack/db"
)

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

// it expects { "description": "Unknown Module Package" } || {"description": <>}
func (s *Server) GetRecipeDescription(w http.ResponseWriter, r *http.Request) error {
	name, err := GetItemFromRequest[string](r)
	if err != nil {
		return err
	}

	idx := s.getPackageIndex(*name)

	desc, err := s.apt.GetRecipeDescription(idx)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(desc); err != nil {
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
	pkgs, err := s.apt.GetAllPackages()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(pkgs); err != nil {
		return err
	}

	return nil
}
