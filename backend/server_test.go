package backend

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/config"
	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/internal/apt"
)

func init() { //nolint:gochecknoinits
	slog.SetDefault(slog.New(slog.DiscardHandler))
}

func TestServer(t *testing.T) {
	conn := filepath.Join(t.TempDir(), "db")

	backend := newBackend(t, conn)
	s := newServer(t, backend)
	backend2 := newBackend(t, conn)

	ch := make(chan bool)
	buildComplete = func() { ch <- true }

	t.Cleanup(func() { buildComplete = func() {} })

	env := db.Environment{
		Name:    "env",
		Path:    "path/to/env",
		Version: 1,
		Tags:    []db.Tag{},
		Packages: []db.Package{
			{Name: "abc"},
		},
	}

	code, resp := getResponse(t, s, "/create-environment", env)
	assertEmptyResp(t, code, resp)

	<-ch

	env2 := db.Environment{
		Name:    "complexEnv",
		Path:    "path/to/complexEnv",
		Version: 1,
		Tags:    []db.Tag{},
		Packages: []db.Package{
			{
				Name:    "r-lib",
				Version: "1.1",
			},
			{
				Name: "python",
			},
		},
	}

	code, resp = getResponse(t, s, "/create-environment", env2)
	assertEmptyResp(t, code, resp)

	<-ch

	t.Log("Close should not exit until building environments have finished")

	assert.NoError(t, backend.Close())
	s.Close()

	s2 := newServer(t, backend2)

	code, resp = getResponse(t, s2, "/get-environments")
	assert.Equal(t, http.StatusOK, code)

	var envs []db.Environment

	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(envs))
	assert.Equal(t, db.Concretised, envs[0].Status)
	assert.Equal(t, db.Concretised, envs[1].Status)

	t.Log("Reconnecting to a database with environments in building status should trigger them to build")

	backend3 := newBackend(t, conn)

	s3 := newHttpServer(t, backend3)
	t.Cleanup(s3.Close)

	assert.NoError(t, backend2.Close())
	s2.Close()

	backend3.updateEnvStatus(&env, db.Building)
	backend3.updateEnvStatus(&env2, db.Building)

	err = backend3.reBuildEnvs()
	assert.NoError(t, err)

	<-ch
	<-ch

	code, resp = getResponse(t, s3, "/get-environments")
	assert.Equal(t, http.StatusOK, code)

	err = json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(envs))
	assert.Equal(t, db.Concretised, envs[0].Status)
	assert.Equal(t, db.Concretised, envs[1].Status)
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newServer(t, nil)
}

func newBackend(t *testing.T, dbConn string) *Server {
	t.Helper()

	root := apt.CreateTestAptRepo(t, apt.ExamplePackages())
	moduleBase := t.TempDir()
	installBase := t.TempDir()
	artefactBase := t.TempDir()

	if dbConn == "" {
		dbConn = filepath.Join(t.TempDir(), "db")
	}

	backend, err := New(&config.Config{
		BaseImgPath:   apt.BuildBase(t),
		ModulePath:    moduleBase,
		TempDir:       "",
		InstallDir:    installBase,
		WrapperScript: "a-wrapper-script",
		AptSrc:        root,
		AptIndexSrc:   filepath.Join(root, "dists", "resolute", "main", "binary-"+runtime.GOARCH, "Packages"),
		ArtefactStore: artefactBase,
		DBConn:        dbConn,
		Driver:        "sqlite3",
	})

	assert.NoError(t, err)

	return backend
}

func newServer(t *testing.T, backend *Server) *httptest.Server {
	t.Helper()

	if backend == nil {
		backend = newBackend(t, "")
	}

	s := httptest.NewServer(backend.Serve())

	t.Cleanup(s.Close)
	t.Cleanup(func() { backend.Close() })

	return s
}

func newHttpServer(t *testing.T, backend *Server) *httptest.Server {
	t.Helper()

	return newServer(t, backend)
}

func getResponse(t *testing.T, s *httptest.Server, endpoint string, body ...any) (int, string) {
	t.Helper()

	var reader io.Reader

	var method string

	if len(body) == 0 {
		method = http.MethodGet
	} else {
		method = http.MethodPost

		switch v := body[0].(type) {
		case io.Reader:
			reader = v
		default:
			jsonBody, err := json.Marshal(body[0])
			assert.NoError(t, err)

			reader = bytes.NewReader(jsonBody)
		}
	}

	r, err := http.NewRequest(method, s.URL+endpoint, reader)
	if err != nil {
		t.Fatalf("getResponse error: %s", err)
	}

	resp, err := s.Client().Do(r)
	if err != nil {
		t.Fatalf("getResponse error: %s", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("getResponse error: %s", err)
	}

	return resp.StatusCode, string(respBody)
}

func assertEmptyResp(t *testing.T, code int, resp string) {
	t.Helper()

	assert.Equal(t, http.StatusNoContent, code)
	assert.Empty(t, resp)
}

func assertBadRequest(t *testing.T, code int, resp string, err error) {
	t.Helper()

	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, resp, err.Error())
}

func checkAllEqual[T db.Environment | db.RecipeRequest](t *testing.T, s *httptest.Server, expected []T) {
	t.Helper()

	var endpoint string

	var v T

	switch any(v).(type) {
	case db.Environment:
		endpoint = "/get-environments"
	case db.RecipeRequest:
		endpoint = "/requested-recipes"
	}

	code, resp := getResponse(t, s, endpoint)
	assert.Equal(t, http.StatusOK, code)

	var actual []T

	err := json.NewDecoder(strings.NewReader(resp)).Decode(&actual)
	assert.NoError(t, err)

	switch any(v).(type) {
	case db.Environment:
		assert.Equal(t,
			expected,
			zeroEnv(t, any(actual).([]db.Environment)), //nolint:errcheck,forcetypeassert
		)
	default:
		assert.Equal(t, expected, actual)
	}
}
