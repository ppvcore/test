package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 2
	config.MaxConnLifetime = 0
	config.MaxConnIdleTime = 5 * 60
	config.HealthCheckPeriod = 60 * time.Second

	var pool *pgxpool.Pool

	// retry loop: ждём пока Postgres станет доступен
	const maxRetries = 10
	wait := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, config)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				// всё ок, возвращаем пул
				return pool, nil
			} else {
				pool.Close()
				err = fmt.Errorf("postgres pool ping failed: %w", pingErr)
			}
		}

		fmt.Printf("Postgres not ready yet (%d/%d): %v\n", i+1, maxRetries, err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
			wait *= 2 // exponential backoff
		}
	}

	return nil, fmt.Errorf("could not connect to postgres after %d retries: %w", maxRetries, err)
}
