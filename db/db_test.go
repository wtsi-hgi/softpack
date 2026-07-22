package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	UniqueConstraintFailed = "UNIQUE constraint failed"
	MissingRequiredFields  = "one or more required fields missing"
)

func TestCreateEnvironments(t *testing.T) {
	ctx, db := setup(t)

	env1 := Environment{
		Name:        "name",
		Path:        "path/to/env",
		Description: "description",
		Version:     1,
		Created:     248933,
		Hidden:      false,
		Packages: []string{
			"pkg1",
			"pkg2",
		},
	}

	err := db.CreateEnvironment(ctx, env1)
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []Environment{env1}, envs)

	env2 := Environment{
		Name:        "name2",
		Path:        "path/to/env2",
		Description: "description",
		Version:     2,
		Created:     24854323,
		Hidden:      false,
		Packages: []string{
			"pkg1",
			"pkg3",
		},
	}

	env3 := Environment{
		Name:        "name3",
		Path:        "path/to/env3",
		Description: "description",
		Version:     3,
		Created:     24854365423,
		Hidden:      true,
		Packages: []string{
			"pkg1",
		},
	}

	err = db.CreateEnvironments(ctx, []Environment{env1, env2})
	assert.ErrorContains(t, err, UniqueConstraintFailed)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, envs, []Environment{env1}) // TODO: Do i want it to still add env2 given that env1 fails?

	err = db.CreateEnvironments(ctx, []Environment{env2, env3})
	assert.NoError(t, err)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, envs, []Environment{env1, env2, env3})

	indexes := []EnvironmentIndex{
		{
			Name:    "name",
			Path:    "path/to/env",
			Version: 1,
		},
		{
			Name:    "name2",
			Path:    "path/to/env2",
			Version: 2,
		},
	}

	envs, err = db.GetEnvironments(ctx, indexes...)
	assert.NoError(t, err)
	assert.Equal(t, envs, []Environment{env1, env2})

	env4 := Environment{
		Path: "path/to/incomplete/env",
	}

	err = db.CreateEnvironment(ctx, env4)
	assert.ErrorContains(t, err, MissingRequiredFields)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, envs, []Environment{env1, env2, env3})
}

func TestUpdateEnvironment(t *testing.T) {
	ctx, db, env := setupWithEnv1(t)

	env.Hidden = true
	env.Description = "new description"

	err := db.UpdateEnvironment(ctx, env)
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx, env.ToIndex())
	assert.NoError(t, err)
	assert.Equal(t, []Environment{env}, envs)
}

func TestDeleteEnvironment(t *testing.T) {
	ctx, db, env := setupWithEnv1(t)

	index := env.ToIndex()

	err := db.DeleteEnvironment(ctx, index)
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx, index)
	assert.ErrorContains(t, err, "record not found")
	assert.Equal(t, len(envs), 0)
}

func TestRequestRecipe(t *testing.T) {
	ctx, db := setup(t)

	r := RecipeRequest{
		Name:    "name",
		Version: "version",
		URL:     "url/for/name",
	}

	err := db.RequestRecipe(ctx, r)
	assert.ErrorContains(t, err, MissingRequiredFields)

	r.Details = "details"

	err = db.RequestRecipe(ctx, r)
	assert.NoError(t, err)

	reqs, err := db.GetRequestedRecipes(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(reqs), 1)
	// checkRecipeEqual(t, reqs[0], r)
	assert.Equal(t, r, reqs[0])
}

func setup(t *testing.T) (context.Context, *DB) {
	t.Helper()

	ctx := t.Context()
	db, err := Connect("sqlite3", ":memory:")
	require.NoError(t, err)

	return ctx, db
}

func setupWithEnv1(t *testing.T) (context.Context, *DB, Environment) {
	ctx, db := setup(t)

	env := Environment{
		Name:        "name",
		Path:        "path/to/env",
		Version:     1,
		Description: "description",
		Created:     248933,
		Hidden:      false,
		Packages: []string{
			"pkg1",
			"pkg2",
		},
	}

	err := db.CreateEnvironment(ctx, env)
	assert.NoError(t, err)

	return ctx, db, env
}

// func checkRecipeEqual(t *testing.T, actual, expected RecipeRequest) {
// 	t.Helper()

// 	actual.ID = expected.ID
// 	assert.Equal(t, actual, expected)
// }
