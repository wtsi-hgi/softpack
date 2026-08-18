package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/wtsi-hgi/softpack/apt"
	"github.com/wtsi-hgi/softpack/config"
	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/utils"
	"vimagination.zapto.org/httpbuffer"
)

var (
	ErrInvalidJSON   = errors.New("invalid json")
	ErrDuplicateItem = errors.New("item to add already exists")
)

type Server struct {
	waitingEnvs utils.WaitingEnvs
	buildTimes  utils.BuildTimes

	db     *db.DB
	apt    *apt.Server
	config *config.Config

	listen net.Listener
}

func (b *Server) Close() error {
	if err := b.listen.Close(); err != nil {
		return err
	}

	for {
		envs, err := b.db.BuildingEnvs(context.Background())
		if err != nil {
			return err
		}

		count := len(envs)

		if count == 0 {
			break
		}

		slog.Info("Waiting for environments to finish building", "count", count)

		time.Sleep(10 * time.Second) //nolint:mnd
	}

	return nil
}

func (b *Server) Serve() http.Handler {
	var m http.ServeMux

	m.Handle("/create-environment", handler(b.CreateEnvironment))
	m.Handle("/get-environments", handler(b.GetEnvironment))
	m.Handle("/delete-environment", handler(b.DeleteEnvironment))
	m.Handle("/add-tag", handler(b.AddEnvironmentTag))
	m.Handle("/set-hidden", handler(b.SetEnvironmentHidden))
	m.Handle("/delete-tag", handler(b.DeleteEnvironmentTag))
	m.Handle("/request-recipe", handler(b.RequestRecipe))
	m.Handle("/requested-recipes", handler(b.GetRequestedRecipes))
	m.Handle("/get-recipe-description", handler(b.GetRecipeDescription))
	m.Handle("/package-collection", handler(b.GetAllPackages))
	m.Handle("/remove-requested-recipe", handler(b.RemoveRequestedRecipe))
	m.Handle("/fulfil-requested-recipe", handler(b.FulfilRequestedRecipe))
	m.Handle("/groups", handler(b.GetGroups))
	m.Handle("/tags", handler(b.GetTags))
	m.Handle("/build-status", handler(b.GetAverageBuildTime))

	return &m
}

type handler func(w http.ResponseWriter, r *http.Request) error

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httpbuffer.Handler{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := h(w, r); err != nil {
				http.Error(w, err.Error(), responseCode(err))
			}
		}),
	}.ServeHTTP(w, r)
}

var httpErrors = []error{
	io.EOF,
	ErrInvalidJSON,
	ErrDuplicateItem,
	db.ErrNoRowsAffected,
	apt.ErrInvalidPackage,
	db.ErrMissingField,
}

func responseCode(err error) int {
	if slices.Contains(httpErrors, err) {
		return http.StatusBadRequest
	}

	if _, ok := errors.AsType[*json.SyntaxError](err); ok { //nolint:errcheck
		return http.StatusBadRequest
	}

	log.Printf("Given error not found in httpErrors %s", err)

	return http.StatusInternalServerError
}

func GetItemFromRequest[T any](r *http.Request) (T, error) {
	var item T

	var buf bytes.Buffer

	n, err := io.Copy(&buf, r.Body)
	if err != nil {
		return item, err
	}

	if n == 0 {
		return item, nil
	}

	if err := json.Unmarshal(buf.Bytes(), &item); err != nil {
		return item, err
	}

	return item, nil
}

func New(config *config.Config) (*Server, error) { //nolint:funlen
	apt, err := apt.New(config.AptIndexSrc, 5*time.Minute) //nolint:mnd
	if err != nil {
		slog.Error("Invalid apt index", "index", config.AptIndexSrc)

		return nil, err
	}

	database, err := db.Connect(config.Driver, config.DBConn)
	if err != nil {
		slog.Error("Failed to connect to database", "driver", config.Driver, "dbconn", config.DBConn)

		return nil, err
	}

	s := &Server{
		db:     database,
		apt:    apt,
		config: config,

		waitingEnvs: utils.NewWaitingEnvs(),
	}

	s.populateServerCaches()

	l, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		slog.Error("Failed to create listener", "err", err)

		return nil, err
	}

	s.listen = l

	return s, nil
}

func (b *Server) Run() error {
	if err := b.reBuildEnvs(); err != nil {
		return err
	}

	return http.Serve(b.listen, b.Serve()) //nolint:gosec
}

func (b *Server) reBuildEnvs() error {
	envs, err := b.db.BuildingEnvs(context.Background())
	if err != nil {
		return err
	}

	count := len(envs)

	if count > 0 {
		slog.Info("environments left in building state, rebuilding...")
	}

	for i, env := range envs {
		slog.Info(fmt.Sprintf("Building environment %d of %d", i, count))
		b.Build(&env)
	}

	return nil
}

func (b *Server) populateServerCaches() { //nolint:gocognit
	ctx := context.Background()

	reqs, _ := b.db.GetRequestedRecipes(ctx) //nolint:errcheck
	envs, _ := b.db.GetEnvironments(ctx)     //nolint:errcheck

	for _, req := range reqs {
		for _, env := range envs {
			if env.Status == db.Concretised {
				if err := b.buildTimes.AddBuildTime(env.BuildStart, env.BuildEnd); err != nil {
					slog.Error("Failure adding build times for env", "env", env.ToIndex())
				}
			}

			for _, pkg := range env.Packages {
				if db.CheckPkgEqual(pkg, req) {
					b.waitingEnvs.Append(req, env)
				}
			}
		}
	}
}
