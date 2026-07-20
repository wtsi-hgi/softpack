package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndGetEnvironments(t *testing.T) {
	ctx, db := Setup(t)

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
	assert.ErrorContains(t, err, "UNIQUE constraint failed")

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, envs, []Environment{env1})

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
}

func TestUpdateEnvironment(t *testing.T) {
	ctx, db := SetupWithEnv1(t)

	index := EnvironmentIndex{
		Name:    "name",
		Path:    "path/to/env",
		Version: 1,
	}

	err := db.UpdateEnvironment(ctx, index, map[string]interface{}{
		"Hidden":      true,
		"Description": "new description",
	})
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx, index)
	assert.NoError(t, err)
	assert.Equal(t, len(envs), 1)
	assert.True(t, envs[0].Hidden)
	assert.Equal(t, envs[0].Description, "new description")
}

func TestDeleteEnvironment(t *testing.T) {
	ctx, db := SetupWithEnv1(t)

	index := EnvironmentIndex{
		Name:    "name",
		Path:    "path/to/env",
		Version: 1,
	}

	err := db.DeleteEnvironment(ctx, index)
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx, index)
	assert.ErrorContains(t, err, "record not found")
	assert.Equal(t, len(envs), 0)
}

func Setup(t *testing.T) (context.Context, *DB) {
	t.Helper()

	ctx := t.Context()
	db, err := Connect("sqlite3", ":memory:")
	require.NoError(t, err)

	return ctx, db
}

func SetupWithEnv1(t *testing.T) (context.Context, *DB) {
	ctx, db := Setup(t)

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

	return ctx, db
}
