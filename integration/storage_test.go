package integration

import (
	"testing"
	"time"

	"github.com/blixenkrone/doodle/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

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
	return fixture{store: store, pool: pool}
}

func TestCreateUser(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	usr, err := f.store.CreateUser(ctx, storage.CreateUserParams{
		UserID: uuid.New(),
		Name:   "dummy",
	})
	require.NoError(t, err)
	require.WithinDuration(t, usr.CreatedAt, time.Now(), time.Second)
}
