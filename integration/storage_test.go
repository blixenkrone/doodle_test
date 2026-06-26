package integration

import (
	"testing"

	"github.com/blixenkrone/doodle/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var actor = storage.Actor{ID: "tester", Role: "doctor"}

type fixture struct {
	store *storage.Store
	pool  *pgxpool.Pool
}

func setup(t *testing.T) fixture {
	t.Helper()
	pg := RunMigrations(t, t.Name())
	t.Cleanup(func() { _ = pg.Teardown() })
	pool := pg.Container()
	store := storage.NewStore(pool)

	ctx := t.Context()
	usr, err := store.Create(ctx, "acme-"+uuid.NewString()[:8], "eu")
	require.NoError(t, err)
	return fixture{store: store, pool: pool}
}
