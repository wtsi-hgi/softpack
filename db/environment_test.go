package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateEnvironments(t *testing.T) {
	ctx, db := setup(t)

	env1 := Environment{
		Name:        "name",        //nolint:goconst
		Path:        "path/to/env", //nolint:goconst
		Description: "description", //nolint:goconst
		Created:     248933,
		Hidden:      false,
		Tags:        []Tag{},
		Packages: []Package{
			{
				Name: NonExistentPkg1,
			},
			{
				Name: NonExistentPkg2,
			},
		},
	}

	err := db.CreateEnvironment(ctx, &env1)
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx)
	assert.NoError(t, err)

	env1.Version = 1
	assert.Equal(t, zeroEnvKey([]Environment{env1}), zeroEnvKey(envs))

	env2 := Environment{
		Name:        "name",
		Path:        "path/to/env",
		Description: "description",
		Created:     24854323,
		Hidden:      false,
		Tags:        []Tag{},
		Packages: []Package{
			{
				Name: NonExistentPkg1,
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
		Created:     24854365423,
		Hidden:      true,
		Tags:        []Tag{},
		Packages: []Package{
			{
				Name: NonExistentPkg1,
			},
		},
	}

	err = db.CreateEnvironments(ctx, []Environment{env2, env3})
	assert.NoError(t, err)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)

	env2.Version = 2
	env3.Version = 1
	assert.Equal(t, zeroEnvKey([]Environment{env1, env2, env3}), zeroEnvKey(envs))

	env4 := Environment{
		Path: "path/to/incomplete/env",
	}

	err = db.CreateEnvironment(ctx, &env4)
	assert.ErrorIs(t, err, ErrMissingField)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, zeroEnvKey([]Environment{env1, env2, env3}), zeroEnvKey(envs))
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

	failureReason := "Failure reason"

	err = db.UpdateStatusWithFailureReason(ctx, UpdateValue[string]{
		EnvironmentIndex: idx,
		Value:            failureReason,
	})
	assert.NoError(t, err)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(envs), 1)
	assert.Equal(t, Failed, envs[0].Status)
	assert.Equal(t, failureReason, envs[0].FailureReason)

	newPkgs := []Package{
		{
			Name:    NonExistentPkg1,
			Version: "1.2",
		},
		{
			Name: "pkg3",
		},
	}

	assert.NoError(t, db.Concretise(env, newPkgs))

	buildStart := int64(1234567)
	buildEnd := int64(7654321)

	assert.NoError(t, db.SetEnvBuildTime(ctx, UpdateValue[int64]{
		EnvironmentIndex: env.ToIndex(),
		Value:            buildStart,
	}, true))

	assert.NoError(t, db.SetEnvBuildTime(ctx, UpdateValue[int64]{
		EnvironmentIndex: env.ToIndex(),
		Value:            buildEnd,
	}, false))

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(envs), 1)
	assert.Equal(t, newPkgs, envs[0].Packages)
	assert.Equal(t, buildStart, envs[0].BuildStart)
	assert.Equal(t, buildEnd, envs[0].BuildEnd)
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

	env2 := Environment{
		Name:        "name",        //nolint:goconst
		Path:        "path/to/env", //nolint:goconst
		Description: "description", //nolint:goconst
		Created:     248933,
		Hidden:      false,
		Tags:        []Tag{},
		Packages: []Package{
			{
				Name: NonExistentPkg1,
			},
			{
				Name: NonExistentPkg2,
			},
		},
	}

	err := db.CreateEnvironment(ctx, &env2)
	assert.NoError(t, err)

	tag := Tag{Name: "newtag"}
	tag2 := Tag{Name: "capy"}

	err = db.AddEnvironmentTag(ctx, UpdateValue[Tag]{
		EnvironmentIndex: idx,
		Value:            tag,
	})
	assert.NoError(t, err)

	err = db.AddEnvironmentTag(ctx, UpdateValue[Tag]{
		EnvironmentIndex: idx,
		Value:            tag2,
	})
	assert.NoError(t, err)

	err = db.AddEnvironmentTag(ctx, UpdateValue[Tag]{
		EnvironmentIndex: env2.ToIndex(),
		Value:            tag2,
	})
	assert.NoError(t, err)

	envs, err := db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(envs), 2)
	assert.Equal(t, []Tag{tag, tag2}, zeroTagKey(envs[0].Tags))
	assert.Equal(t, []Tag{tag2}, zeroTagKey(envs[1].Tags))

	tags, err := db.GetTags(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []Tag{tag, tag2, tag2}, zeroTagKey(tags))

	err = db.DeleteEnvironmentTag(ctx, UpdateValue[Tag]{
		EnvironmentIndex: idx,
		Value:            tag,
	})
	assert.NoError(t, err)

	envs, err = db.GetEnvironments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(envs))
	assert.Equal(t, []Tag{tag2}, zeroTagKey(envs[0].Tags))
	assert.Equal(t, []Tag{tag2}, zeroTagKey(envs[1].Tags))

	tags, err = db.GetTags(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []Tag{tag2, tag2}, zeroTagKey(tags))
}

func TestGetBuildingEnvs(t *testing.T) {
	ctx, db := setup(t)

	envs, err := db.GetBuildingEnvs(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []Environment{}, envs)

	env1 := Environment{
		Name:     "env",
		Path:     "path/to/env",
		Version:  1,
		Packages: []Package{},

		Status: Building,
	}

	assert.NoError(t, db.CreateEnvironment(ctx, &env1))

	env2 := Environment{
		Name:     "env2",
		Path:     "path/to/env2",
		Version:  2,
		Packages: []Package{},

		Status: Building,
	}

	assert.NoError(t, db.CreateEnvironment(ctx, &env2))

	envs, err = db.GetBuildingEnvs(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []Environment{env1, env2}, envs)
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
