package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Up applies migrations from the supplied directory. Call only in explicitly
// approved environments; the application server does not run migrations.
func Up(ctx context.Context, databaseURL, migrationsDir string) error {
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer conn.Close()

	if err := conn.PingContext(ctx); err != nil {
		return fmt.Errorf("ping migration database: %w", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, conn, migrationsDir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
