package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const runtimeRole = "naeos_intelligence_runtime"

// OpenPool creates a bounded pool with conservative defaults. The caller owns Close.
// It fails closed unless the connection authenticates as the dedicated runtime role.
func OpenPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = make(map[string]string)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = "naeos_intelligence, pg_catalog"
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
	var currentUser string
	if err := pool.QueryRow(ctx, "SELECT current_user").Scan(&currentUser); err != nil {
		pool.Close()
		return nil, fmt.Errorf("verify PostgreSQL runtime role: %w", err)
	}
	if currentUser != runtimeRole {
		pool.Close()
		return nil, fmt.Errorf("PostgreSQL runtime role must be %q, got %q", runtimeRole, currentUser)
	}

	return pool, nil
}
