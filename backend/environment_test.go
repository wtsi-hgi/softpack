package backend

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
)

const MissingRequiredFields = "one or more required fields missing"

func TestCreateEnvironment(t *testing.T) {
	s := New()
	var envs []db.Environment

	code, resp := getResponse(t, s.GetEnvironment, "/get-environment")
	assert.Equal(t, 200, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{}, envs)

	invalid := db.Environment{
		Name: "invalid",
		Path: "path/to/invalid",
	}
	code, resp = getResponse(t, s.CreateEnvironment, "/create-environment", invalid)
	assert.Equal(t, 500, code)
	assert.Contains(t, resp, MissingRequiredFields)

	code, resp = getResponse(t, s.GetEnvironment, "/get-environment")
	assert.Equal(t, 200, code)
	err = json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{}, envs)

	environment := db.Environment{
		Name:        "test",
		Path:        "path/to/test",
		Version:     1,
		Description: "description",
		Created:     1,
	}
	code, resp = getResponse(t, s.CreateEnvironment, "/create-environment", environment)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s.GetEnvironment, "/get-environment")
	assert.Equal(t, 200, code)
	err = json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{environment}, envs)
}

func TestDeleteEnvironment(t *testing.T) {
	s := setupWithEnv(t)
	var envs []db.Environment

	idx := db.EnvironmentIndex{
		Name:    "test",
		Path:    "path/to/test",
		Version: 1,
	}

	code, resp := getResponse(t, s.DeleteEnvironment, "/delete-environment", idx)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s.GetEnvironment, "/get-environment")
	assert.Equal(t, 200, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{}, envs)
}

func TestUpdateEnvironment(t *testing.T) {
	s := setupWithEnv(t)
	var envs []db.Environment

	env := db.Environment{
		Name:        "test",
		Path:        "path/to/test",
		Version:     1,
		Description: "description",
		Created:     1,
	}
	idx := env.ToIndex()

	code, resp := getResponse(t, s.UpdateEnvironment, "/update-environment", idx)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s.GetEnvironment, "/get-environment", idx)
	assert.Equal(t, 200, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{env}, envs)
}

func setupWithEnv(t *testing.T) *Server {
	t.Helper()

	s := New()
	var envs []db.Environment

	environment := db.Environment{
		Name:        "test",
		Path:        "path/to/test",
		Version:     1,
		Description: "description",
		Created:     1,
	}
	code, resp := getResponse(t, s.CreateEnvironment, "/create-environment", environment)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s.GetEnvironment, "/get-environment")
	assert.Equal(t, 200, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{environment}, envs)

	return s
}
