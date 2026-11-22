// Package testdb provisions an isolated PostgreSQL identity model for integration tests only.
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

type Database struct {
	AdminDSN    string
	RuntimeDSN  string
	MigratorDSN string
}

const (
	runtimeLogin = "naeos_intelligence_runtime_login"
	migratorLogin = "naeos_intelligence_migrator_login"
)

func Bootstrap(ctx context.Context) (*Database, error) {
	adminDSN := os.Getenv("NAEOS_TEST_ADMIN_DATABASE_URL")
	if adminDSN == "" {
		adminDSN = os.Getenv("DATABASE_URL")
	}
	if adminDSN == "" {
		return nil, fmt.Errorf("NAEOS_TEST_ADMIN_DATABASE_URL is required for PostgreSQL integration tests")
	}
	runtimePassword := os.Getenv("NAEOS_TEST_RUNTIME_PASSWORD")
	if runtimePassword == "" {
		runtimePassword = "naeos-runtime-test"
	}
	migratorPassword := os.Getenv("NAEOS_TEST_MIGRATOR_PASSWORD")
	if migratorPassword == "" {
		migratorPassword = "naeos-migrator-test"
	}

	conn, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		return nil, fmt.Errorf("connect to ephemeral PostgreSQL admin database: %w", err)
	}
	defer conn.Close(ctx)

	statements := []string{
		`DO $bootstrap$
		BEGIN
		  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='naeos_intelligence_runtime') THEN
		    CREATE ROLE naeos_intelligence_runtime NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
		  END IF;
		  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='naeos_intelligence_migrator') THEN
		    CREATE ROLE naeos_intelligence_migrator NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
		  END IF;
		  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='naeos_intelligence_runtime_login') THEN
		    CREATE ROLE naeos_intelligence_runtime_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
		  END IF;
		  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='naeos_intelligence_migrator_login') THEN
		    CREATE ROLE naeos_intelligence_migrator_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
		  END IF;
		END
		$bootstrap$`,
		`ALTER ROLE naeos_intelligence_runtime NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS`,
		`ALTER ROLE naeos_intelligence_migrator NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS`,
		fmt.Sprintf("ALTER ROLE naeos_intelligence_runtime_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD %s", quoteLiteral(runtimePassword)),
		fmt.Sprintf("ALTER ROLE naeos_intelligence_migrator_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD %s", quoteLiteral(migratorPassword)),
		`GRANT naeos_intelligence_runtime TO naeos_intelligence_runtime_login`,
		`GRANT naeos_intelligence_migrator TO naeos_intelligence_migrator_login`,
		`CREATE SCHEMA IF NOT EXISTS naeos_intelligence AUTHORIZATION naeos_intelligence_migrator`,
		`ALTER SCHEMA naeos_intelligence OWNER TO naeos_intelligence_migrator`,
		`GRANT USAGE ON SCHEMA naeos_intelligence TO naeos_intelligence_runtime`,
		`GRANT CREATE ON SCHEMA naeos_intelligence TO naeos_intelligence_migrator`,
		`REVOKE CREATE ON SCHEMA naeos_intelligence FROM naeos_intelligence_runtime`,
		`CREATE SCHEMA IF NOT EXISTS naeos_control_plane`,
		`CREATE SCHEMA IF NOT EXISTS naeos_identity`,
		`CREATE TABLE IF NOT EXISTS public.crm_guard_test (id integer PRIMARY KEY)`,
		`CREATE TABLE IF NOT EXISTS naeos_control_plane.guard_test (id integer PRIMARY KEY)`,
		`CREATE TABLE IF NOT EXISTS naeos_identity.guard_test (id integer PRIMARY KEY)`,
		`REVOKE ALL ON public.crm_guard_test FROM PUBLIC, naeos_intelligence_runtime, naeos_intelligence_migrator`,
		`REVOKE ALL ON naeos_control_plane.guard_test FROM PUBLIC, naeos_intelligence_runtime, naeos_intelligence_migrator`,
		`REVOKE ALL ON naeos_identity.guard_test FROM PUBLIC, naeos_intelligence_runtime, naeos_intelligence_migrator`,
		`REVOKE ALL ON SCHEMA naeos_control_plane FROM naeos_intelligence_runtime, naeos_intelligence_migrator`,
		`REVOKE ALL ON SCHEMA naeos_identity FROM naeos_intelligence_runtime, naeos_intelligence_migrator`,
		`ALTER ROLE naeos_intelligence_runtime_login SET search_path TO 'pg_catalog, naeos_intelligence'`,
		`ALTER ROLE naeos_intelligence_migrator_login SET search_path TO 'pg_catalog, naeos_intelligence'`,
	}
	for _, statement := range statements {
		if _, err := conn.Exec(ctx, statement); err != nil {
			return nil, fmt.Errorf("bootstrap isolated PostgreSQL test roles/schema: %w", err)
		}
	}

	runtimeDSN, err := replaceCredentials(adminDSN, runtimeLogin, runtimePassword)
	if err != nil {
		return nil, fmt.Errorf("build runtime test DSN: %w", err)
	}
	migratorDSN, err := replaceCredentials(adminDSN, migratorLogin, migratorPassword)
	if err != nil {
		return nil, fmt.Errorf("build migrator test DSN: %w", err)
	}
	return &Database{AdminDSN: adminDSN, RuntimeDSN: runtimeDSN, MigratorDSN: migratorDSN}, nil
}

func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func replaceCredentials(dsn, username, password string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return "", fmt.Errorf("PostgreSQL integration DSN must use a postgres URL")
	}
	u.User = url.UserPassword(username, password)
	return u.String(), nil
}
