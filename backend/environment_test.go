package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
	"gorm.io/gorm"
)

const (
	NonExistentPkg1 = "nonexistentpkg1"
	NonExistentPkg2 = "nonexistentpkg2"
)

func TestCreateEnvironment(t *testing.T) {
	s := newTestServer(t)

	checkAllEqual(t, s, []db.Environment{})

	invalid := db.Environment{
		Name: "invalid",
		Path: "path/to/invalid",
	}
	code, resp := getResponse(t, s, "/create-environment", invalid)
	assertBadRequest(t, code, resp, db.ErrMissingField)

	checkAllEqual(t, s, []db.Environment{})

	env := db.Environment{
		Name:        "test",         //nolint: goconst
		Path:        "path/to/test", //nolint: goconst
		Description: "description",  //nolint: goconst
		Tags:        []db.Tag{},
		Packages: []db.Package{
			{
				Name: NonExistentPkg1,
			},
			{
				Name: NonExistentPkg2,
			},
		},
	}
	code, resp = getResponse(t, s, "/create-environment", env)
	assertEmptyResp(t, code, resp)

	env.Status = db.Building
	env.Version = "1"
	env.Created = 0
	checkAllEqual(t, s, zeroEnv(t, []db.Environment{env}))

	req := db.RecipeRequest{
		Name:      "new-pkg",
		Version:   "1",
		URL:       "path/to/new-pkg",
		Details:   "desc",
		Requester: "sky",
	}

	code, resp = getResponse(t, s, "/request-recipe", req)
	assertEmptyResp(t, code, resp)

	waitingEnv := db.Environment{
		Name:        "waiting",         //nolint: goconst
		Path:        "path/to/waiting", //nolint: goconst
		Description: "description",     //nolint: goconst
		Tags:        []db.Tag{},
		Packages: []db.Package{
			{
				Name: "new-pkg",
			},
		},
	}
	code, resp = getResponse(t, s, "/create-environment", waitingEnv)
	assertEmptyResp(t, code, resp)

	waitingEnv.Status = db.Waiting
	waitingEnv.Version = "1"
	waitingEnv.Created = 0
	checkAllEqual(t, s, zeroEnv(t, []db.Environment{env, waitingEnv}))
}

func TestDeleteEnvironment(t *testing.T) {
	s, env := setupWithEnv(t)

	idx := env.ToIndex()

	code, resp := getResponse(t, s, "/delete-environment", idx)
	assertEmptyResp(t, code, resp)

	checkAllEqual(t, s, []db.Environment{})

	code, resp = getResponse(t, s, "/delete-environment", idx)
	assertBadRequest(t, code, resp, db.ErrNoRowsAffected)
}

func TestUpdateEnvironment(t *testing.T) {
	s, env := setupWithEnv(t)

	var envs []db.Environment

	idx := env.ToIndex()

	code, resp := getResponse(t, s, "/set-hidden", db.UpdateValue[bool]{
		EnvironmentIndex: idx,
		Value:            true,
	})
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/get-environments")
	assert.Equal(t, 200, code)

	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(envs))
	assert.True(t, envs[0].Hidden)
}

func TestAddAndDeleteTags(t *testing.T) {
	s, env := setupWithEnv(t)

	env2 := db.Environment{
		Name:        "env2",
		Path:        "path/to/env2",
		Version:     "1",
		Description: "desc",
		Tags:        []db.Tag{},
		Packages: []db.Package{
			{
				Name: NonExistentPkg1,
			},
		},
	}

	tag := "new tag"

	code, resp := getResponse(t, s, "/add-tag", db.UpdateValue[string]{
		EnvironmentIndex: env2.ToIndex(),
		Value:            tag,
	})
	assertBadRequest(t, code, resp, gorm.ErrRecordNotFound)

	code, resp = getResponse(t, s, "/create-environment", &env2)
	assertEmptyResp(t, code, resp)

	u := db.UpdateValue[string]{
		EnvironmentIndex: env.ToIndex(),
		Value:            tag,
	}

	code, resp = getResponse(t, s, "/delete-tag", u)
	assertBadRequest(t, code, resp, db.ErrNoRowsAffected)

	code, resp = getResponse(t, s, "/add-tag", u)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/add-tag", u)
	assertBadRequest(t, code, resp, ErrDuplicateItem)

	env.Tags = []db.Tag{db.Tag{Name: tag}}

	env2.Status = db.Building
	checkAllEqual(t, s, zeroEnv(t, []db.Environment{env, env2}))

	code, resp = getResponse(t, s, "/tags")
	assert.Equal(t, http.StatusOK, code)

	var tags []db.Tag

	err := json.NewDecoder(strings.NewReader(resp)).Decode(&tags)
	assert.NoError(t, err)
	assert.Equal(t, []db.Tag{db.Tag{Name: tag}}, zeroTagKey(t, tags))

	code, resp = getResponse(t, s, "/delete-tag", u)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/tags")
	assert.Equal(t, http.StatusOK, code)

	err = json.NewDecoder(strings.NewReader(resp)).Decode(&tags)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(tags))
}

func setupWithEnv(t *testing.T) (*httptest.Server, db.Environment) {
	t.Helper()

	s := newTestServer(t)

	env := db.Environment{
		Name:        "test",
		Path:        "path/to/test",
		Description: "description",
		Tags:        []db.Tag{},
		Status:      db.Building,
		Packages: []db.Package{
			{
				Name: NonExistentPkg1,
			},
			{
				Name: NonExistentPkg2,
			},
		},
	}
	code, resp := getResponse(t, s, "/create-environment", env)
	assertEmptyResp(t, code, resp)

	env.Version = "1"
	checkAllEqual(t, s, []db.Environment{env})

	return s, env
}

func zeroTagKey(t *testing.T, tags []db.Tag) []db.Tag { // TODO: I dont like this duplication
	t.Helper()

	for n := range tags {
		tags[n].ID = 0
	}

	return tags
}

func zeroEnv(t *testing.T, envs []db.Environment) []db.Environment {
	t.Helper()

	for n, env := range envs {
		envs[n].ID = 0
		envs[n].Created = 0
		envs[n].BuildStart = 0
		envs[n].Readme = ""

		for t := range env.Tags {
			env.Tags[t].ID = 0
		}
	}

	return envs
}
