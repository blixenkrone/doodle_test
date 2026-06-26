package integration

import (
	"os"
	"testing"

	"github.com/blixenkrone/sdk/docker"

	"github.com/blixenkrone/doodle/internal/storage/sql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dockerPool docker.Pool

func TestMain(m *testing.M) {
	dpool, err := docker.NewPool()
	if err != nil {
		panic(err)
	}
	dockerPool = dpool
	os.Exit(m.Run())
}

// RunMigrations spins up a throwaway postgres container and applies the embedded
// schema migrations against it. The returned resource is cleaned up via t.Cleanup.
func RunMigrations(t *testing.T, workflowName string) *docker.Resource[*pgxpool.Pool] {
	pgResource, err := dockerPool.Postgres(t.Context(), workflowName)
	require.NoError(t, err)
	pool := pgResource.Container()

	require.NoError(t, pool.Ping(t.Context()))

	driver, err := iofs.New(sql.Migrations, "migrations")
	require.NoError(t, err)

	migrator, err := migrate.NewWithSourceInstance("iofs", driver, pool.Config().ConnString())
	assert.NoError(t, err)
	t.Cleanup(func() {
		if serr, err := migrator.Close(); serr != nil || err != nil {
			t.Fatalf("source: %v - driver: %v", serr, err)
		}
	})
	require.NoError(t, migrator.Up())

	return pgResource
}
