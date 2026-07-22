package backend

import (
	"encoding/json"
	"net/http"

	"github.com/wtsi-hgi/softpack/db"
)

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
