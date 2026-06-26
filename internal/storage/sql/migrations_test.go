package sql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/blixenkrone/sdk/docker"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationsUpDown attempts to apply all 'up' migrations and then revert them with 'down'.
func TestMigrationsUpDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Second*20)
	defer cancel()

	dockerPool, err := docker.NewPool()
	if err != nil {
		panic(err)
	}

	pgResource, err := dockerPool.Postgres(ctx, "postgres_test_db")
	if err != nil {
		panic(err)
	}
	t.Cleanup(func() {
		if err := pgResource.Teardown(); err != nil {
			panic(err)
		}
	})
	pgPool := pgResource.Container()

	driver, err := iofs.New(Migrations, "migrations")
	assert.NoError(t, err)
	defer func() {
		if err := driver.Close(); err != nil {
			panic(err)
		}
	}()

	// Create migrate instance using the iofs source driver and database DSN
	dbDSN := pgPool.Config().ConnString()
	t.Logf("migration to dsn: %s", dbDSN)
	m, err := migrate.NewWithSourceInstance("iofs", driver, dbDSN)
	assert.NoError(t, err)
	defer func() {
		if serr, err := m.Close(); serr != nil || err != nil {
			msg := fmt.Sprintf("source: %v - driver: %s", serr, err)
			panic(msg)
		}
	}()

	err = m.Up()
	assert.NoError(t, err)
	// Check for dirty migrations
	version, dirty, err := m.Version()
	require.NoError(t, err)
	require.False(t, dirty)
	require.Greater(t, version, uint(0))

	err = m.Down()
	assert.NoError(t, err)
}
