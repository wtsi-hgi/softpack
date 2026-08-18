package backend

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/apt"
	"github.com/wtsi-hgi/softpack/db"
)

func TestRequestRecipe(t *testing.T) {
	s := newTestServer(t)

	req := db.RecipeRequest{
		Name:    "name",
		Version: "version",      //nolint: goconst
		URL:     "url/for/name", //nolint: goconst
	}

	code, resp := getResponse(t, s, "/request-recipe", req)
	assertBadRequest(t, code, resp, db.ErrMissingField)

	req.Details = "details" //nolint: goconst
	code, resp = getResponse(t, s, "/request-recipe", req)
	assertEmptyResp(t, code, resp)

	checkAllEqual(t, s, []db.RecipeRequest{req})
}

func TestGetRecipeDescription(t *testing.T) {
	s := newTestServer(t)

	code, resp := getResponse(t, s, "/get-recipe-description", "pkg5")
	assertBadRequest(t, code, resp, apt.ErrInvalidPackage)

	code, resp = getResponse(t, s, "/get-recipe-description", "abc")
	assert.Equal(t, http.StatusOK, code)

	var desc string

	err := json.NewDecoder(strings.NewReader(resp)).Decode(&desc)
	assert.NoError(t, err)
	assert.Equal(t, "desc1", desc)

	reqPkgDesc := "description of requested package"

	req := db.RecipeRequest{
		Name:      "requested-pkg",
		Version:   "1",
		URL:       "path/to/requested-pkg",
		Details:   reqPkgDesc,
		Requester: "sky",
	}

	code, resp = getResponse(t, s, "/request-recipe", req)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/get-recipe-description", "*requested-pkg")
	assert.Equal(t, http.StatusOK, code)

	err = json.NewDecoder(strings.NewReader(resp)).Decode(&desc)
	assert.NoError(t, err)
	assert.Equal(t, reqPkgDesc, desc)
}

func TestGetAllPackages(t *testing.T) {
	s := newTestServer(t)

	code, resp := getResponse(t, s, "/package-collection")
	assert.Equal(t, http.StatusOK, code)

	expectedPackages := []apt.Package{
		{
			Name:     "abc", //nolint:goconst
			Versions: []string{"1", "2"},
		},
		{
			Name:     "py-xyz",
			Versions: []string{"2.1"},
		},
		{
			Name:     "python", //nolint:goconst
			Versions: []string{"3.13"},
		},
		{
			Name:     "r",
			Versions: []string{"4.4.0"},
		},
		{
			Name:     "r-lib",
			Versions: []string{"1.1"},
		},
	}

	var pkgs []apt.Package

	err := json.NewDecoder(strings.NewReader(resp)).Decode(&pkgs)
	assert.NoError(t, err)
	assert.Equal(t, expectedPackages, pkgs)
}

func TestRemoveRequestedRecipe(t *testing.T) {
	s := newTestServer(t)

	code, resp := getResponse(t, s, "/remove-requested-recipe", db.RecipeRequest{})
	assertBadRequest(t, code, resp, db.ErrMissingField)

	req := db.RecipeRequest{
		Name:    "name",
		Version: "version",
		URL:     "url/for/name",
		Details: "details",
	}

	code, resp = getResponse(t, s, "/request-recipe", req)
	assertEmptyResp(t, code, resp)

	checkAllEqual(t, s, []db.RecipeRequest{req})

	code, resp = getResponse(t, s, "/remove-requested-recipe", req)
	assertEmptyResp(t, code, resp)

	checkAllEqual(t, s, []db.RecipeRequest{})
}

func TestFulfilRequestedRecipe(t *testing.T) {
	s := newTestServer(t)

	req := db.RecipeRequest{
		Name:    "requestedpkg",
		Version: "version",
		URL:     "url/for/name",
		Details: "details",
	}
	code, resp := getResponse(t, s, "/request-recipe", req)
	assertEmptyResp(t, code, resp)

	req2 := db.RecipeRequest{
		Name:    "requestedpkg2",
		Version: "version",
		URL:     "url/for/name",
		Details: "details",
	}
	code, resp = getResponse(t, s, "/request-recipe", req2)
	assertEmptyResp(t, code, resp)

	env := db.Environment{
		Name:        "test",         //nolint: goconst
		Path:        "path/to/test", //nolint: goconst
		Description: "description",  //nolint: goconst
		Tags:        []db.Tag{},
		Packages: []db.Package{
			{
				Name: "requestedpkg",
			},
			{
				Name:    "requestedpkg2",
				Version: "version",
			},
		},
	}
	code, resp = getResponse(t, s, "/create-environment", env)
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/get-environments")
	assert.Equal(t, http.StatusOK, code)

	var envs []db.Environment

	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(envs))
	assert.Equal(t, db.Waiting, envs[0].Status)

	code, resp = getResponse(t, s, "/fulfil-requested-recipe", db.FulfilRequestBody{
		RecipeRequest:    req,
		CanonicalName:    "abc",
		CanonicalVersion: "1",
	})
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/get-environments")
	assert.Equal(t, http.StatusOK, code)

	err = json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(envs))
	assert.Equal(t, db.Waiting, envs[0].Status)

	code, resp = getResponse(t, s, "/fulfil-requested-recipe", db.FulfilRequestBody{
		RecipeRequest:    req2,
		CanonicalName:    "abc",
		CanonicalVersion: "2",
	})
	assertEmptyResp(t, code, resp)

	code, resp = getResponse(t, s, "/get-environments")
	assert.Equal(t, http.StatusOK, code)

	err = json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(envs))
	assert.Equal(t, db.Building, envs[0].Status)
}
