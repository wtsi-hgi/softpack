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
		Version: "version",
		URL:     "url/for/name",
	}

	code, resp := getResponse(t, s, "/request-recipe", req)
	assertBadRequest(t, code, resp, db.ErrMissingField)

	req.Details = "details"
	code, resp = getResponse(t, s, "/request-recipe", req)
	assertEmptyResp(t, code, resp)

	checkAllEqual(t, s, []db.RecipeRequest{req})
}

func TestGetRecipeDescription(t *testing.T) {
	s := newTestServer(t)

	code, resp := getResponse(t, s, "/get-recipe-description", "pkg5")
	assertBadRequest(t, code, resp, apt.ErrInvalidPackage)

	code, resp = getResponse(t, s, "/get-recipe-description", "pkg1")
	assert.Equal(t, http.StatusOK, code)

	var desc RecipeDescriptionResponse

	err := json.NewDecoder(strings.NewReader(resp)).Decode(&desc)
	assert.NoError(t, err)
	assert.Equal(t, "desc1", desc.Description)
}

func TestGetAllPackages(t *testing.T) {
	s := newTestServer(t)

	code, resp := getResponse(t, s, "/package-collection")
	assert.Equal(t, http.StatusOK, code)

	expectedPackages := []apt.Package{
		{
			Name:     "pkg1", //nolint:goconst
			Versions: []string{"1", "2"},
		},
		{
			Name:     "pkg2", //nolint:goconst
			Versions: []string{"2", "5", "8"},
		},
		{
			Name:     "pkg3",
			Versions: []string{"3"},
		},
		{
			Name:     "pkg4",
			Versions: []string{"4"},
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

// func TestFulfilRequestedRecipe(t *testing.T) {
// 	s := newTestServer(t)

// 	req := db.RecipeRequest{
// 		Name:    "requestedpkg",
// 		Version: "version",
// 		URL:     "url/for/name",
// 		Details: "details",
// 	}
// 	code, resp := getResponse(t, s, "/request-recipe", req)
// 	assertEmptyResp(t, code, resp)

// 	req2 := db.RecipeRequest{
// 		Name:    "requestedpkg2",
// 		Version: "version",
// 		URL:     "url/for/name",
// 		Details: "details",
// 	}
// 	code, resp = getResponse(t, s, "/request-recipe", req2)
// 	assertEmptyResp(t, code, resp)

// 	env := db.Environment{
// 		Name:        "test",
// 		Path:        "path/to/test",
// 		Version:     1,
// 		Description: "description",
// 		Created:     1,
// 		Packages: []db.Package{
// 			{
// 				Name:     "requestedpkg",
// 				Versions: []string{"version"},
// 			},
// 			{
// 				Name:     "requestedpkg2",
// 				Versions: []string{"version"},
// 			},
// 		},
// 	}
// 	code, resp = getResponse(t, s, "/create-environment", env)
// 	assertEmptyResp(t, code, resp)

// 	checkAllEqual(t, s, []db.Environment{})

// 	code, resp = getResponse(t, s, "/fulfil-requested-recipe", req)
// 	assertEmptyResp(t, code, resp)

// 	checkAllEqual(t, s, []db.Environment{})

// 	code, resp = getResponse(t, s, "/fulfil-requested-recipe", req2)
// 	assertEmptyResp(t, code, resp)

// 	checkAllEqual(t, s, []db.Environment{env})
// }
