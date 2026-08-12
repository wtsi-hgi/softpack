package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
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
		Status:      db.Building,
		Tags:        []db.Tag{},
		Packages: []db.Package{
			{
				Name: "pkg1",
			},
			{
				Name: "pkg2",
			},
		},
	}
	code, resp = getResponse(t, s, "/create-environment", env)
	assertEmptyResp(t, code, resp)

	env.Version = 1
	env.Created = 0
	checkAllEqual(t, s, zeroEnv(t, []db.Environment{env}))
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

	tag := db.Tag{Name: "new tag"}

	u := db.UpdateValue[db.Tag]{
		EnvironmentIndex: env.ToIndex(),
		Value:            tag,
	}

	code, resp := getResponse(t, s, "/delete-tag", u)
	assertBadRequest(t, code, resp, db.ErrNoRowsAffected)

	code, resp = getResponse(t, s, "/add-tag", u)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/add-tag", u)
	assertBadRequest(t, code, resp, ErrDuplicateItem)

	env.Tags = []db.Tag{tag}

	checkAllEqual(t, s, zeroEnv(t, []db.Environment{env}))

	code, resp = getResponse(t, s, "/tags")
	assert.Equal(t, http.StatusOK, code)

	var tags []db.Tag

	err := json.NewDecoder(strings.NewReader(resp)).Decode(&tags)
	assert.NoError(t, err)
	assert.Equal(t, []db.Tag{tag}, zeroTagKey(t, tags))
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
				Name: "pkg1",
			},
			{
				Name: "pkg2",
			},
		},
	}
	code, resp := getResponse(t, s, "/create-environment", env)
	assertEmptyResp(t, code, resp)

	env.Version = 1
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

		for t := range env.Tags {
			env.Tags[t].ID = 0
		}
	}

	return envs
}
