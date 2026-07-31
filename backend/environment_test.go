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
		Name:        "test",
		Path:        "path/to/test",
		Version:     1,
		Description: "description",
		Created:     1,
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

	checkAllEqual(t, s, zeroEnvKey([]db.Environment{env}))
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

	tags := []db.Tag{{Name: "tag1"}, {Name: "tag2"}}

	u := db.UpdateEnv{
		EnvironmentIndex: env.ToIndex(),
		Description:      ptrTo("new description"),
		Hidden:           ptrTo(true),
		Tags:             ptrTo(tags),
	}

	code, resp := getResponse(t, s, "/update-environment", u)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/get-environments", env.ToIndex())
	assert.Equal(t, 200, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(envs))
	assert.Equal(t, "new description", envs[0].Description)
	assert.True(t, envs[0].Hidden)
	assert.Equal(t, tags, zeroTagKey(envs[0].Tags))
}

func TestAddAndDeleteTags(t *testing.T) {
	s, env := setupWithEnv(t)

	tag := db.Tag{Name: "new tag"}

	u := db.UpdateValue{
		EnvironmentIndex: env.ToIndex(),
		Value:            tag.Name,
	}

	code, resp := getResponse(t, s, "/delete-tag", u)
	assertBadRequest(t, code, resp, db.ErrNoRowsAffected)

	code, resp = getResponse(t, s, "/add-tag", u)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/add-tag", u)
	assertBadRequest(t, code, resp, ErrDuplicateItem)

	env.Tags = []db.Tag{tag}

	checkAllEqual(t, s, zeroEnvKey([]db.Environment{env}))

	code, resp = getResponse(t, s, "/tags")
	assert.Equal(t, http.StatusOK, code)
	var tags []db.Tag
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&tags)
	assert.NoError(t, err)
	assert.Equal(t, []db.Tag{tag}, zeroTagKey(tags))
}

func setupWithEnv(t *testing.T) (*httptest.Server, db.Environment) {
	s := newTestServer(t)

	environment := db.Environment{
		Name:        "test",
		Path:        "path/to/test",
		Version:     1,
		Description: "description",
		Created:     1,
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
	code, resp := getResponse(t, s, "/create-environment", environment)
	assertEmptyResp(t, code, resp)

	checkAllEqual(t, s, []db.Environment{environment})

	return s, environment
}

// TODO: Add a test where an environment is created, requiring a requested recipie
// verify that it is not queued for build.
// then add the recipie, verify that the environment dependent on it is built
//
// potentially add multiple envs with multiple recipie dependencies to be more thorough

func zeroTagKey(tags []db.Tag) []db.Tag { // TODO: I dont like this duplication
	for n := range tags {
		tags[n].ID = 0
	}

	return tags
}

func zeroEnvKey(envs []db.Environment) []db.Environment {
	for n, env := range envs {
		envs[n].ID = 0

		for t := range env.Tags {
			env.Tags[t].ID = 0
		}
	}

	return envs
}
