package postgres

import (
	"context"
	"fmt"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

type DB struct {
	pool *pgxpool.Pool
}

func NewFromConn(conn *pgxpool.Pool) DB {
	return DB{conn}
}

type ConnConfig struct {
	User,
	Password,
	Host,
	Port,
	DBName,
	SSLMode string
	PoolMaxConns int
}

func (c ConnConfig) BuildDSNConnStr() string {
	vars := []any{c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode, c.PoolMaxConns}
	return fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s pool_max_conns=%d", vars...)
}

func NewFromConnectionString(ctx context.Context, connStr, serviceName string) (DB, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return DB{}, fmt.Errorf("connetion string error: %w", err)
	}

	// Add global tags to all database spans

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return DB{}, fmt.Errorf("error connecting to DB: %w", err)
	}
	return DB{pool}, nil
}

func (s DB) Pool() *pgxpool.Pool {
	return s.pool
}

func (s DB) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s DB) Close(ctx context.Context) {
	s.pool.Close()
}
