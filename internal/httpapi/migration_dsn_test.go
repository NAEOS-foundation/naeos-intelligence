package httpapi_test

import (
	"os"
	"testing"
)

func migrationDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("MIGRATOR_DATABASE_URL")
	if dsn == "" {
		t.Fatal("MIGRATOR_DATABASE_URL must be set for PostgreSQL integration tests")
	}
	return dsn
}
