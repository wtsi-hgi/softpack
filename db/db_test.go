package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	UniqueConstraintFailed = "UNIQUE constraint failed"
	NonExistentPkg1        = "nonexistentpkg1"
	NonExistentPkg2        = "nonexistentpkg2"
)

func setup(t *testing.T) (context.Context, *DB) {
	t.Helper()

	ctx := t.Context()
	db, err := Connect("sqlite3", ":memory:")
	require.NoError(t, err)

	return ctx, db
}

func setupWithEnv1(t *testing.T) (context.Context, *DB, Environment) {
	t.Helper()

	ctx, db := setup(t)

	env := Environment{
		Name:        "name",        //nolint:goconst
		Path:        "path/to/env", //nolint:goconst
		Description: "description", //nolint:goconst
		Created:     248933,
		Hidden:      false,
		Tags:        []Tag{},
		Packages: []Package{
			{
				Name: NonExistentPkg1, //nolint:goconst
			},
			{
				Name: NonExistentPkg2,
			},
		},
	}

	err := db.CreateEnvironment(ctx, &env)
	assert.NoError(t, err)

	return ctx, db, env
}
