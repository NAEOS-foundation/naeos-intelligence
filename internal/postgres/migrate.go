package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const intelligenceSchema = "naeos_intelligence"
const migratorRole = "naeos_intelligence_migrator"

// Up applies migrations from the supplied directory using the dedicated
// migrator role. The target schema must exist and the authenticated role
// must match; the function refuses to fall back to public or runtime access.
func Up(ctx context.Context, databaseURL, migrationsDir string) error {
	cfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parse migration database config: %w", err)
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = make(map[string]string)
	}
	cfg.RuntimeParams["search_path"] = intelligenceSchema + ", pg_catalog"

	conn := stdlib.OpenDB(*cfg)
	defer conn.Close()

	if err := conn.PingContext(ctx); err != nil {
		return fmt.Errorf("ping migration database: %w", err)
	}
	var currentUser string
	if err := conn.QueryRowContext(ctx, "SELECT current_user").Scan(&currentUser); err != nil {
		return fmt.Errorf("verify migration role: %w", err)
	}
	if currentUser != migratorRole {
		return fmt.Errorf("PostgreSQL migrator role must be %q, got %q", migratorRole, currentUser)
	}

	var schemaExists bool
	if err := conn.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = $1)", intelligenceSchema).Scan(&schemaExists); err != nil {
		return fmt.Errorf("verify target schema: %w", err)
	}
	if !schemaExists {
		return fmt.Errorf("target schema %q is not provisioned; refusing to run migrations", intelligenceSchema)
	}

	goose.SetTableName(intelligenceSchema + ".goose_db_version")
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, conn, migrationsDir); err != nil {
		return fmt.Errorf("apply migrations in schema %q: %w", intelligenceSchema, err)
	}
	if _, err := conn.ExecContext(ctx, "REVOKE ALL ON TABLE "+intelligenceSchema+".goose_db_version FROM "+runtimeRole); err != nil {
		return fmt.Errorf("revoke runtime access to migration ledger: %w", err)
	}
	return nil
}
