package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const UniqueConstraintFailed = "UNIQUE constraint failed"

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
		Version:     1,
		Description: "description", //nolint:goconst
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

	err := db.CreateEnvironment(ctx, env)
	assert.NoError(t, err)

	return ctx, db, env
}
