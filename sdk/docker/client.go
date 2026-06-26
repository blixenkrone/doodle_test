package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

type Pool struct {
	pool *dockertest.Pool
}

func NewPool() (Pool, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return Pool{}, err
	}
	pool.MaxWait = 120 * time.Second
	return Pool{pool}, nil
}

type Resource[T any] struct {
	r         *dockertest.Resource
	container T
}

func (p Pool) Postgres(ctx context.Context, dbname string) (*Resource[*pgxpool.Pool], error) {
	env := []string{
		"POSTGRES_USER=admin",
		"POSTGRES_PASSWORD=password",
		fmt.Sprintf("POSTGRES_DB=%s", dbname),
		"listen_addresses = '*'",
	}

	runOpts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14-alpine",
		Env:        env,
	}
	config := func(cfg *docker.HostConfig) {
		cfg.AutoRemove = true
		cfg.RestartPolicy = docker.RestartPolicy{Name: "no"}
	}
	resource, err := p.pool.RunWithOptions(&runOpts, config)
	if err != nil {
		return nil, fmt.Errorf("error starting pg resource: %w", err)
	}
	// Tell docker to hard kill the container in 120 seconds
	if err := resource.Expire(120); err != nil {
		return nil, fmt.Errorf("resource expiry: %w", err)
	}

	var pgdb *pgxpool.Pool
	initFn := func() error {
		hostAndPort := resource.GetHostPort("5432/tcp")
		connStr := fmt.Sprintf("postgres://admin:password@%s/%s?sslmode=disable", hostAndPort, dbname)
		conn, err := pgxpool.New(ctx, connStr)
		if err != nil {
			return fmt.Errorf("error creating postgres store: %w", err)
		}
		pgdb = conn
		return pgdb.Ping(ctx)
	}

	if err := p.pool.Retry(initFn); err != nil {
		return nil, err
	}

	return &Resource[*pgxpool.Pool]{resource, pgdb}, nil
}

func (r Resource[T]) Teardown() error {
	return r.r.Close()
}

func (r Resource[T]) Container() T {
	return r.container
}
