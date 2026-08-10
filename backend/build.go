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
	s.updateEnvStatus(env, db.Building, nil) //nolint:errcheck

	go func() {
		defer func() {
			if err := s.buildTimes.AddBuildTime(env.BuildStart, env.BuildEnd); err != nil {
				slog.Error("Failure adding build times for env", "env", env.ToIndex())
			}

			buildComplete()
		}()

		env.BuildStart = time.Now().Unix()

		slog.Debug("starting build", "env", env.Name, "time", env.BuildStart)

		artefacts, err := install.Install(s.config, *env)
		if err != nil {
			slog.Error("Install", "error", err)
			s.updateEnvStatus(env, db.Failed, err) //nolint:errcheck

			return
		}

		if err := s.db.Concretise(*env, artefacts.Packages); err != nil {
			slog.Error("Conretise", "error", err)
			s.updateEnvStatus(env, db.Failed, err) //nolint:errcheck

			return
		}

		slog.Debug("build finished", "env", env.Name, "time", env.BuildStart)

		env.BuildEnd = time.Now().Unix()

		s.updateEnvStatus(env, db.Concretised, nil) //nolint:errcheck
	}()
}

func (s *Server) updateEnvStatus(env *db.Environment, status db.Status, err error) error {
	if status == db.Failed {
		env.FailureReason = err.Error()
	}

	return s.db.UpdateStatus(context.Background(), db.UpdateValue[db.Status]{
		EnvironmentIndex: env.ToIndex(),
		Value:            status,
	})
}

func (s *Server) GetAverageBuildTime(w http.ResponseWriter, _ *http.Request) error {
	avrg := s.buildTimes.GetAverageBuildTime()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(avrg); err != nil {
		return err
	}

	return nil
}
