package backend

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/install"
)

var buildComplete = func() {}

func (s *Server) Build(env *db.Environment) {
	s.updateEnvStatus(env, db.Building) //nolint:errcheck

	go func() {
		defer func() {
			delete(s.buildingEnvs, env.ID)
			buildComplete()
		}()

		env.BuildStart = time.Now().Unix()

		artefacts, err := install.Install(s.config, *env)
		if err != nil {
			slog.Error("Install", "error", err)
			s.updateEnvStatus(env, db.Failed) //nolint:errcheck

			return
		}

		if err := s.db.Concretise(*env, artefacts.Packages); err != nil {
			slog.Error("Conretise", "error", err)
			s.updateEnvStatus(env, db.Failed) //nolint:errcheck

			return
		}

		s.updateEnvStatus(env, db.Concretised) //nolint:errcheck
	}()
}

// TODO: Should probs do in a transaction so the db and map cant become out of sync.
func (s *Server) updateEnvStatus(env *db.Environment, status int) error {
	if err := s.db.UpdateEnvironment(context.Background(), db.UpdateEnv{
		EnvironmentIndex: env.ToIndex(),
		Status:           ptrTo(status),
	}); err != nil {
		return err
	}

	env.Status = status
	s.buildingEnvs[env.ID] = env

	return nil
}

func (s *Server) GetAverageBuildTime(w http.ResponseWriter, _ *http.Request) error {
	totalBuildTime := int64(0)

	for _, env := range s.buildingEnvs {
		totalBuildTime += time.Now().Unix() - env.BuildStart
	}

	avrg := totalBuildTime / int64(len(s.buildingEnvs))

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(avrg); err != nil {
		return err
	}

	return nil
}
