package backend

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/config"
	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/internal/apt"
)

func TestGetAverageBuildTime(t *testing.T) {
	apt.SkipIfBadEnvironment(t)

	src := apt.CreateTestAptRepo(t, apt.ExamplePackages())

	backend := New(&config.Config{
		AptSrc:        src,
		AptIndexSrc:   filepath.Join(src, "dists", "resolute", "main", "binary-"+runtime.GOARCH, "Packages"),
		BaseImgPath:   apt.BuildBase,
		InstallDir:    t.TempDir(),
		WrapperScript: "capy",
		ModulePath:    t.TempDir(),
		ArtefactStore: t.TempDir(),
		Driver:        "sqlite3",
	})
	s := newHttpServer(t, backend)

	ch := make(chan bool)
	buildComplete = func() { ch <- true }

	t.Cleanup(func() { buildComplete = func() {} })

	env1 := &db.Environment{
		Name:    "env1",
		Path:    "/path/to/env1",
		Version: 1,
		Created: 1234,
		Tags:    []db.Tag{},
		Packages: []db.Package{
			{
				Name: "nonexistentpkg",
			},
		},
	}

	code, resp := getResponse(t, s, "/create-environment", env1)
	assertEmptyResp(t, code, resp)

	e, exists := backend.buildingEnvs[env1.ID]
	assert.True(t, exists)
	assert.Equal(t, db.Building, e.Status)

	code, resp = getResponse(t, s, "/get-environments", env1.ToIndex())
	assert.Equal(t, http.StatusOK, code)

	var envs []db.Environment

	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(envs))
	assert.Equal(t, db.Building, envs[0].Status)

	<-ch

	_, exists = backend.buildingEnvs[env1.ID]
	assert.False(t, exists)

	code, resp = getResponse(t, s, "/get-environments", env1.ToIndex())
	assert.Equal(t, http.StatusOK, code)

	err = json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(envs))
	assert.Equal(t, db.Failed, envs[0].Status)

	// backend.config = &config.Config{
	// 	AptSrc:        apt.CreateTestAptRepo(t, apt.ExamplePackages()),
	// 	BaseImgPath:   apt.BuildBase,
	// 	InstallDir:    t.TempDir(),
	// 	WrapperScript: "capy",
	// 	ModulePath:    t.TempDir(),
	// 	ArtefactStore: t.TempDir(),
	// }

	env2 := &db.Environment{
		Name:    "env2",
		Path:    "/path/to/env2",
		Version: 2,
		Created: 2,
		Tags:    []db.Tag{},
		Packages: []db.Package{
			{
				Name:    "abc",
				Version: "2",
			},
			{
				Name: "python",
			},
		},
	}

	t.Log("Force env2 to build by calling create environment")

	code, resp = getResponse(t, s, "/create-environment", env2)
	assertEmptyResp(t, code, resp)

	<-ch

	_, exists = backend.buildingEnvs[env2.ID]
	assert.False(t, exists)

	code, resp = getResponse(t, s, "/get-environments", env1.ToIndex())
	assert.Equal(t, http.StatusOK, code)

	err = json.NewDecoder(strings.NewReader(resp)).Decode(&envs)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(envs))
	assert.Equal(t, db.Concretised, envs[1].Status)

	assert.Equal(t, []db.Package{
		{
			Name:    "abc",
			Version: "2",
		},
		{
			Name:    "python",
			Version: "3.13",
		},
	}, envs[1].Packages)
}
