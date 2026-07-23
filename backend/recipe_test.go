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
	assertBadRequest(t, code, resp, apt.ErrPackageNotFound)

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
			Name: "pkg1",
			Versions: []string{
				"1",
				"2",
			},
			Description: "desc1",
		},
		{
			Name: "pkg2",
			Versions: []string{
				"2",
				"5",
				"8",
			},
			Description: "desc2",
		},
		{
			Name: "pkg3",
			Versions: []string{
				"3",
			},
			Description: "desc3",
		},
		{
			Name: "pkg4",
			Versions: []string{
				"4",
			},
			Description: "",
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
	assertBadRequest(t, code, resp, db.ErrMissingItem)

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
