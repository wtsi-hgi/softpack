package backend

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
)

var ErrMissingRequiredField = Error{
	err:  errors.New("one or more required fields missing"),
	code: http.StatusBadRequest,
}

func TestCreateEnvironment(t *testing.T) {
	s := New()

	checkAllEnvsEqual(t, s, []db.Environment{})

	invalid := db.Environment{
		Name: "invalid",
		Path: "path/to/invalid",
	}
	code, resp := getResponse(t, s.CreateEnvironment, "/create-environment", invalid)
	assertHasError(t, code, resp, ErrMissingRequiredField)

	checkAllEnvsEqual(t, s, []db.Environment{})

	environment := db.Environment{
		Name:        "test",
		Path:        "path/to/test",
		Version:     1,
		Description: "description",
		Created:     1,
	}
	code, resp = getResponse(t, s.CreateEnvironment, "/create-environment", environment)
	assertEmptyResp(t, code, resp)

	checkAllEnvsEqual(t, s, []db.Environment{environment})
}

func TestDeleteEnvironment(t *testing.T) {
	s, env := setupWithEnv(t)

	idx := env.ToIndex()

	code, resp := getResponse(t, s.DeleteEnvironment, "/delete-environment", idx)
	assertEmptyResp(t, code, resp)

	checkAllEnvsEqual(t, s, []db.Environment{})
}

func TestUpdateEnvironment(t *testing.T) {
	s, env := setupWithEnv(t)
	var envs []db.Environment

	env.Description = "new description"
	env.Hidden = true
	idx := env.ToIndex()

	code, resp := getResponse(t, s.UpdateEnvironment, "/update-environment", env)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s.GetEnvironment, "/get-environments", idx)
	assert.Equal(t, 200, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{env}, envs)
}

func TestAddAndDeleteTags(t *testing.T) {
	s, env := setupWithEnv(t)

	u := db.UpdateByIndex{
		EnvironmentIndex: env.ToIndex(),
		Value:            "new tag",
	}

	code, resp := getResponse(t, s.DeleteEnvironmentTag, "/delete-tag", u)
	assertHasError(t, code, resp, ErrMissingItem)

	code, resp = getResponse(t, s.AddEnvironmentTag, "/add-tag", u)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s.AddEnvironmentTag, "/add-tag", u)
	assertHasError(t, code, resp, ErrDuplicateItem)

	env.Tags = []string{"new tag"}

	checkAllEnvsEqual(t, s, []db.Environment{env})
}

func TestToggleHidden(t *testing.T) {
	s, env := setupWithEnv(t)

	code, resp := getResponse(t, s.ToggleEnvironmentHidden, "/set-hidden", env)
	assertEmptyResp(t, code, resp)

	var actual []db.Environment

	code, resp = getResponse(t, s.GetEnvironment, "/get-environments")
	assert.Equal(t, http.StatusOK, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&actual)
	assert.NoError(t, err)

	assert.NotEqual(t, actual[0].Hidden, env.Hidden)
}

func setupWithEnv(t *testing.T) (*Server, db.Environment) {
	t.Helper()

	s := New()

	environment := db.Environment{
		Name:        "test",
		Path:        "path/to/test",
		Version:     1,
		Description: "description",
		Created:     1,
	}
	code, resp := getResponse(t, s.CreateEnvironment, "/create-environment", environment)
	assertEmptyResp(t, code, resp)

	checkAllEnvsEqual(t, s, []db.Environment{environment})

	return s, environment
}

func checkAllEnvsEqual(t *testing.T, s *Server, expected []db.Environment) {
	t.Helper()

	var actual []db.Environment

	code, resp := getResponse(t, s.GetEnvironment, "/get-environments")
	assert.Equal(t, http.StatusOK, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
