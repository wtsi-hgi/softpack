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

	environment := db.Environment{
		Name:        "test",
		Path:        "path/to/test",
		Version:     1,
		Description: "description",
		Created:     1,
	}
	code, resp = getResponse(t, s, "/create-environment", environment)
	assertEmptyResp(t, code, resp)

	checkAllEqual(t, s, []db.Environment{environment})
}

func TestDeleteEnvironment(t *testing.T) {
	s, env := setupWithEnv(t)

	idx := env.ToIndex()

	code, resp := getResponse(t, s, "/delete-environment", idx)
	assertEmptyResp(t, code, resp)

	checkAllEqual(t, s, []db.Environment{})
}

func TestUpdateEnvironment(t *testing.T) {
	s, env := setupWithEnv(t)
	var envs []db.Environment

	env.Description = "new description"
	env.Hidden = true
	idx := env.ToIndex()

	code, resp := getResponse(t, s, "/update-environment", env)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/get-environments", idx)
	assert.Equal(t, 200, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{env}, envs)

	// TODO: Should probably test tags/hidden status are updated correctly using this method
}

func TestAddAndDeleteTags(t *testing.T) {
	s, env := setupWithEnv(t)

	u := db.UpdateByIndex{
		EnvironmentIndex: env.ToIndex(),
		Value:            "new tag",
	}

	code, resp := getResponse(t, s, "/delete-tag", u)
	assertBadRequest(t, code, resp, ErrMissingItem)

	code, resp = getResponse(t, s, "/add-tag", u)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/add-tag", u)
	assertBadRequest(t, code, resp, ErrDuplicateItem)

	env.Tags = []string{"new tag"}

	checkAllEqual(t, s, []db.Environment{env})
}

func TestToggleHidden(t *testing.T) {
	s, env := setupWithEnv(t)

	code, resp := getResponse(t, s, "/set-hidden", env)
	assertEmptyResp(t, code, resp)

	var actual []db.Environment

	code, resp = getResponse(t, s, "/get-environments")
	assert.Equal(t, http.StatusOK, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&actual)
	assert.NoError(t, err)

	assert.NotEqual(t, actual[0].Hidden, env.Hidden)
}

func setupWithEnv(t *testing.T) (*httptest.Server, db.Environment) {
	s := newTestServer(t)

	environment := db.Environment{
		Name:        "test",
		Path:        "path/to/test",
		Version:     1,
		Description: "description",
		Created:     1,
	}
	code, resp := getResponse(t, s, "/create-environment", environment)
	assertEmptyResp(t, code, resp)

	checkAllEqual(t, s, []db.Environment{environment})

	return s, environment
}
