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
