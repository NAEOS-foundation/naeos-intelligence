package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenPool creates a bounded pool with conservative defaults. The caller owns Close.
func OpenPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.ConnConfig.RuntimeParams == nil { cfg.ConnConfig.RuntimeParams = make(map[string]string) }
	cfg.ConnConfig.RuntimeParams["search_path"] = "pg_catalog, naeos_intelligence"
	cfg.MaxConns = 12
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
