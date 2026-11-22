package postgres

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRuntimeRolePrivilegeBoundary(t *testing.T) {
	if os.Getenv("NAEOS_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set NAEOS_POSTGRES_INTEGRATION=1 to run PostgreSQL role isolation test")
	}
	runtimeDSN := os.Getenv("DATABASE_URL")
	if runtimeDSN == "" {
		t.Fatal("DATABASE_URL must be set to the runtime role DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := Up(ctx, migrationDSN(t), "../../migrations"); err != nil {
		t.Fatalf("apply migrations as migrator: %v", err)
	}
	pool, err := OpenPool(ctx, runtimeDSN)
	if err != nil {
		t.Fatalf("open runtime pool: %v", err)
	}
	defer pool.Close()

	var currentUser string
	if err := pool.QueryRow(ctx, "SELECT current_user").Scan(&currentUser); err != nil {
		t.Fatal(err)
	}
	if currentUser != runtimeRole {
		t.Fatalf("authenticated role = %q, want %q", currentUser, runtimeRole)
	}

	if _, err := pool.Exec(ctx, "CREATE TABLE naeos_intelligence.runtime_ddl_probe(id integer)"); err == nil {
		t.Fatal("runtime role unexpectedly created schema object")
	}
	if _, err := pool.Exec(ctx, "SELECT * FROM naeos_intelligence.goose_db_version"); err == nil {
		t.Fatal("runtime role unexpectedly read migration ledger")
	}
	if _, err := pool.Exec(ctx, "DELETE FROM naeos_intelligence.intelligence_requests WHERE false"); err == nil {
		t.Fatal("runtime role unexpectedly had DELETE privilege")
	}
}

func TestRuntimeRoleCannotApplyMigrations(t *testing.T) {
	if os.Getenv("NAEOS_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set NAEOS_POSTGRES_INTEGRATION=1 to run PostgreSQL role isolation test")
	}
	runtimeDSN := os.Getenv("DATABASE_URL")
	if runtimeDSN == "" {
		t.Fatal("DATABASE_URL must be set to runtime DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := Up(ctx, runtimeDSN, "../../migrations"); err == nil {
		t.Fatal("runtime credentials unexpectedly accepted for migrations")
	}
}

func TestMigratorRoleCannotOpenRuntimePool(t *testing.T) {
	if os.Getenv("NAEOS_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set NAEOS_POSTGRES_INTEGRATION=1 to run PostgreSQL role isolation test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := OpenPool(ctx, migrationDSN(t)); err == nil {
		t.Fatal("migrator credentials unexpectedly accepted for runtime pool")
	}
}
