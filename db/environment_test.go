package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		Tags:        []Tag{},
		Packages: []Package{
			{
				Name: "pkg1",
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
	assert.Equal(t, []Environment{env1, env2}, zeroEnvKey(envs))

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

	u := UpdateEnv{
		EnvironmentIndex: env.ToIndex(),
		Hidden:           ptrTo(false),
		Description:      ptrTo("new new description"),
	}

	err := db.UpdateEnvironment(ctx, u)
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx, env.ToIndex())
	assert.NoError(t, err)
	assert.Equal(t, len(envs), 1)
	assert.False(t, envs[0].Hidden)
	assert.Equal(t, envs[0].Description, "new new description")

	tags := []Tag{{Name: "new tag"}}

	u = UpdateEnv{
		EnvironmentIndex: env.ToIndex(),
		Tags:             ptrTo(tags),
	}

	err = db.UpdateEnvironment(ctx, u)
	assert.NoError(t, err)

	envs, err = db.GetEnvironments(ctx, env.ToIndex())
	assert.NoError(t, err)
	assert.Equal(t, len(envs), 1)
	assert.Equal(t, tags, envs[0].Tags)

	err = db.UpdateEnvironment(ctx, UpdateEnv{EnvironmentIndex: env.ToIndex()})
	assert.NoError(t, err)

	envs, err = db.GetEnvironments(ctx, env.ToIndex())
	assert.NoError(t, err)
	assert.Equal(t, len(envs), 1)
	assert.Equal(t, envs[0].Tags, tags)
}

func TestDeleteEnvironment(t *testing.T) {
	ctx, db, env := setupWithEnv1(t)

	index := env.ToIndex()

	err := db.DeleteEnvironment(ctx, index)
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx, index)
	assert.ErrorContains(t, err, "record not found")
	assert.Equal(t, len(envs), 0)

	err = db.DeleteEnvironment(ctx, index)
	assert.ErrorIs(t, err, ErrNoRowsAffected)
}

func TestAddAndDeleteTags(t *testing.T) {
	ctx, db, env := setupWithEnv1(t)
	index := env.ToIndex()

	uidx := UpdateValue{
		EnvironmentIndex: index,
		Value:            "newtag",
	}

	err := db.AddEnvironmentTag(ctx, uidx)
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx, env.ToIndex())
	assert.NoError(t, err)
	assert.Equal(t, []Tag{{Name: "newtag"}}, zeroTagKey(envs[0].Tags))
}

func zeroTagKey(tags []Tag) []Tag {
	for n := range tags {
		tags[n].ID = 0
	}

	return tags
}

func zeroEnvKey(envs []Environment) []Environment {
	for n, _ := range envs {
		envs[n].ID = 0

		// for t := range env.Tags {
		// 	env.Tags[t].ID = 0
		// }
	}

	return envs
}
