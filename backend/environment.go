package backend

import (
	"encoding/json"
	"net/http"

	"github.com/wtsi-hgi/softpack/db"
)

func (s *Server) CreateEnvironment(w http.ResponseWriter, r *http.Request) {
	s.envMu.Lock()
	defer s.envMu.Unlock()

	var env db.Environment

	err := json.NewDecoder(r.Body).Decode(&env)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	err = s.db.CreateEnvironment(r.Context(), env)
	if err != nil {
		http.Error(w, "db grab failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) GetEnvironment(w http.ResponseWriter, r *http.Request) {
	s.envMu.RLock()

	envs, err := s.db.GetEnvironments(r.Context())
	if err != nil {
		http.Error(w, "db grab failed", http.StatusInternalServerError)
		return
	}

	s.envMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(envs)
	if err != nil {
		http.Error(w, "json encoding failed", http.StatusInternalServerError)
	}
}

func (s *Server) DeleteEnvironment(w http.ResponseWriter, r *http.Request)       {}
func (s *Server) AddEnvironmentTag(w http.ResponseWriter, r *http.Request)       {}
func (s *Server) DeleteEnvironmentTag(w http.ResponseWriter, r *http.Request)    {}
func (s *Server) ToggleEnvironmentHidden(w http.ResponseWriter, r *http.Request) {}
