package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestRecipe(t *testing.T) {
	ctx, db := setup(t)

	r := RecipeRequest{
		Name:    "name",
		Version: "version",
		URL:     "url/for/name",
	}

	err := db.RequestRecipe(ctx, r)
	assert.ErrorIs(t, err, ErrMissingField)

	r.Details = "details"

	err = db.RequestRecipe(ctx, r)
	assert.NoError(t, err)

	reqs, err := db.GetRequestedRecipes(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(reqs), 1)
	assert.Equal(t, r, reqs[0])
}

func TestRemoveRequestedRecipe(t *testing.T) {
	ctx, db := setup(t)

	r := RecipeRequest{
		Name:    "name",
		Version: "version",
		URL:     "url/for/name",
		Details: "details",
	}

	err := db.RemoveRequestedRecipe(ctx, r)
	assert.ErrorIs(t, err, ErrMissingItem)

	err = db.RequestRecipe(ctx, r)
	assert.NoError(t, err)

	reqs, err := db.GetRequestedRecipes(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(reqs), 1)
	assert.Equal(t, r, reqs[0])

	err = db.RemoveRequestedRecipe(ctx, r)
	assert.NoError(t, err)

	reqs, err = db.GetRequestedRecipes(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(reqs), 0)
}
