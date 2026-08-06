package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateEnvironments(t *testing.T) {
	ctx, db := setup(t)

	env1 := Environment{
		Name:        "name", //nolint:goconst
		Path:        "path/to/env",
		Description: "description", //nolint:goconst
		Version:     1,
		Created:     248933,
		Hidden:      false,
		Tags:        []Tag{},
		Packages: []Package{
			{
				Name: "pkg1", //nolint:goconst
			},
			{
				Name: "pkg2",
			},
		},
	}

	err := db.CreateEnvironment(ctx, env1)
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []Environment{env1}, zeroEnvKey(envs))

	env2 := Environment{
		Name:        "name2",
		Path:        "path/to/env2",
		Description: "description",
		Version:     2,
		Created:     24854323,
		Hidden:      false,
		Tags:        []Tag{},
		Packages: []Package{
			{
				Name: "pkg1",
			},
			{
				Name: "pkg3",
			},
		},
	}

	env3 := Environment{
		Name:        "name3",
		Path:        "path/to/env3",
		Description: "description",
		Version:     3,
		Created:     24854365423,
		Hidden:      true,
		Tags:        []Tag{},
		Packages: []Package{
			{
				Name: "pkg1",
			},
		},
	}

	err = db.CreateEnvironments(ctx, []Environment{env1, env2})
	assert.ErrorContains(t, err, UniqueConstraintFailed)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []Environment{env1}, zeroEnvKey(envs)) // TODO: Do i want it to still add env2 given that env1 fails?

	err = db.CreateEnvironments(ctx, []Environment{env2, env3})
	assert.NoError(t, err)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []Environment{env1, env2, env3}, zeroEnvKey(envs))

	env4 := Environment{
		Path: "path/to/incomplete/env",
	}

	err = db.CreateEnvironment(ctx, env4)
	assert.ErrorIs(t, err, ErrMissingField)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, zeroEnvKey(envs), []Environment{env1, env2, env3})
}

func ptrTo[T any](v T) *T {
	return &v
}

func TestUpdateEnvironment(t *testing.T) {
	ctx, db, env := setupWithEnv1(t)
	idx := env.ToIndex()

	err := db.UpdateHidden(ctx, UpdateValue[bool]{
		EnvironmentIndex: idx,
		Value:            false,
	})
	assert.NoError(t, err)

	err = db.UpdateStatus(ctx, UpdateValue[Status]{
		EnvironmentIndex: idx,
		Value:            Concretised,
	})
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(envs), 1)
	assert.False(t, envs[0].Hidden)
	assert.Equal(t, Concretised, envs[0].Status)
}

func TestDeleteEnvironment(t *testing.T) {
	ctx, db, env := setupWithEnv1(t)

	index := env.ToIndex()

	err := db.DeleteEnvironment(ctx, index)
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx)
	assert.Equal(t, len(envs), 0)
	assert.NoError(t, err)

	err = db.DeleteEnvironment(ctx, index)
	assert.ErrorIs(t, err, ErrNoRowsAffected)
}

func TestAddAndDeleteTags(t *testing.T) {
	ctx, db, env := setupWithEnv1(t)
	idx := env.ToIndex()

	tag := Tag{Name: "newtag"}

	err := db.AddEnvironmentTag(ctx, UpdateValue[Tag]{
		EnvironmentIndex: idx,
		Value:            tag,
	})
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(envs), 1)
	assert.Equal(t, []Tag{tag}, zeroTagKey(envs[0].Tags))

	err = db.DeleteEnvironmentTag(ctx, UpdateValue[Tag]{
		EnvironmentIndex: idx,
		Value:            tag,
	})
	assert.NoError(t, err)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []Tag{}, zeroTagKey(envs[0].Tags))
}

func zeroTagKey(tags []Tag) []Tag {
	for n := range tags {
		tags[n].ID = 0
	}

	return tags
}

func zeroEnvKey(envs []Environment) []Environment {
	for n := range envs {
		envs[n].ID = 0
	}

	return envs
}
