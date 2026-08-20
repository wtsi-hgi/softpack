package backend

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/wtsi-hgi/softpack/build"
	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/install"
)

var buildComplete = func() {}

func (s *Server) Build(env *db.Environment) {
	s.updateEnvStatus(env, db.Building)

	go s.buildEnv(env)
}

func (s *Server) buildEnv(env *db.Environment) {
	var (
		artefacts *build.Artefacts
		err       error
	)

	defer s.deferred(env, &err, artefacts)

	env.BuildStart = time.Now().Unix()
	s.setEnvBuildTime(env, env.BuildStart, true)

	slog.Debug("starting build", "env", env.Name, "time", env.BuildStart)

	artefacts, err = install.Install(s.config, *env)
	if err != nil {
		slog.Error("Install failure", "env", env, "error", err, "log", artefacts.Log)

		return
	}

	if err = s.db.Concretise(*env, artefacts.Packages); err != nil {
		slog.Error("Failure concretising db packages", "env", env, "error", err)

		return
	}

	slog.Debug("build finished", "env", env.Name, "time", env.BuildStart)

	env.BuildEnd = time.Now().Unix()
	s.updateEnvStatus(env, db.Concretised)
	s.setEnvBuildTime(env, env.BuildEnd, false)
}

func (s *Server) deferred(env *db.Environment, err *error, a *build.Artefacts) {
	defer buildComplete()
	defer func() {
		if err := s.SendBuildStatusEmail(env); err != nil {
			slog.Error("Failure sending environment build status email.")
		}
	}()

	if *err != nil {
		var log string
		if a != nil {
			log = a.Log
		}

		slog.Error("Build", "error", *err)
		s.setEnvFailed(env, *err, log)

		return
	}

	if err := s.buildTimes.AddBuildTime(env.BuildStart, env.BuildEnd); err != nil {
		slog.Error("Failure adding build times for env", "env", env.ToIndex())
	}
}

func (s *Server) updateEnvStatus(env *db.Environment, status db.Status) {
	if err := s.db.UpdateStatus(context.Background(), db.UpdateValue[db.Status]{
		EnvironmentIndex: env.ToIndex(),
		Value:            status,
	}); err != nil {
		slog.Error("Failure setting Status for env", "env", env, "status", status)
	}
}

func (s *Server) setEnvFailed(env *db.Environment, err error, log string) {
	env.FailureReason = err.Error() + log

	if err := s.db.UpdateStatusWithFailureReason(context.Background(), db.UpdateValue[string]{
		EnvironmentIndex: env.ToIndex(),
		Value:            log,
	}); err != nil {
		slog.Error("Failure setting fail Status for env", "env", env, "failureReason", "")
	}
}

func (s *Server) setEnvBuildTime(env *db.Environment, time int64, start bool) {
	if err := s.db.SetEnvBuildTime(context.Background(), db.UpdateValue[int64]{
		EnvironmentIndex: env.ToIndex(),
		Value:            time,
	}, start); err != nil {
		slog.Error("Failure setting BuildEnd time for env", "env", env, "start", start)
	}
}

func (s *Server) GetAverageBuildTime(w http.ResponseWriter, _ *http.Request) error {
	avrg := s.buildTimes.GetAverageBuildTime()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(avrg); err != nil {
		return err
	}

	return nil
}
