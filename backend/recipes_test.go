package backend

import (
	"testing"

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
	assertHasError(t, code, resp, ErrMissingRequiredField)

	req.Details = "details"
	code, resp = getResponse(t, s, "/request-recipe", req)
	assertEmptyResp(t, code, resp)

	checkAllEqual(t, s, []db.RecipeRequest{req})
}
