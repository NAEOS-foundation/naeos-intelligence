package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Up applies migrations from the supplied directory using a dedicated,
// schema-scoped migrator identity. The application server never runs migrations.
func Up(ctx context.Context, databaseURL, migrationsDir string) error {
	cfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parse PostgreSQL migrator URL: %w", err)
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = make(map[string]string)
	}
	cfg.RuntimeParams["search_path"] = migratorSearchPath
	cfg.RuntimeParams["role"] = migratorGroupRole
	conn := stdlib.OpenDB(*cfg)
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	if err := conn.PingContext(ctx); err != nil {
		return fmt.Errorf("ping migration database: %w", err)
	}
	if err := verifyMigratorRole(ctx, conn); err != nil {
		return err
	}
	if err := rejectLegacyPublicGooseLedger(ctx, conn); err != nil {
		return err
	}

	goose.SetTableName("naeos_intelligence.goose_db_version")
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, conn, migrationsDir); err != nil {
		return fmt.Errorf("apply Intelligence schema migrations: %w", err)
	}
	return nil
}

func rejectLegacyPublicGooseLedger(ctx context.Context, conn *sql.DB) error {
	var exists bool
	if err := conn.QueryRowContext(ctx, "SELECT to_regclass('public.goose_db_version') IS NOT NULL").Scan(&exists); err != nil {
		return fmt.Errorf("inspect legacy Goose ledger: %w", err)
	}
	if exists {
		return errors.New("legacy public.goose_db_version detected; stop migration and perform a separately reviewed ledger/schema migration")
	}
	return nil
}

func verifyMigratorRole(ctx context.Context, conn *sql.DB) error {
	var (
		sessionUser, currentUser, schemaOwner, searchPath string
		loginCanLogin, loginSuper, loginCreateDB, loginCreateRole, loginBypass, loginInherit bool
		isMember, groupCanLogin, groupSuper, groupCreateDB, groupCreateRole, groupBypass, groupHasParent bool
		schemaExists, schemaUsage, schemaCreate, controlPlaneUsage, identityUsage, publicCreate bool
		externalTableAccess, externalSequenceAccess, externalSecurityDefinerAccess, publicLedgerExists bool
		membershipCount int
	)
	err := conn.QueryRowContext(ctx, `
		SELECT session_user::text, current_user::text,
		       login.rolcanlogin, login.rolsuper, login.rolcreatedb, login.rolcreaterole, login.rolbypassrls, login.rolinherit,
		       pg_has_role(session_user, 'naeos_intelligence_migrator', 'MEMBER'),
		       (SELECT count(*) FROM pg_auth_members WHERE member=login.oid),
		       migrator_group.rolcanlogin, migrator_group.rolsuper, migrator_group.rolcreatedb,
		       migrator_group.rolcreaterole, migrator_group.rolbypassrls,
		       EXISTS (SELECT 1 FROM pg_auth_members WHERE member=migrator_group.oid),
		       n.oid IS NOT NULL, COALESCE(has_schema_privilege(current_user,'naeos_intelligence','USAGE'),false),
		       COALESCE(has_schema_privilege(current_user,'naeos_intelligence','CREATE'),false),
		       COALESCE(has_schema_privilege(current_user,'naeos_control_plane','USAGE'),false),
		       COALESCE(has_schema_privilege(current_user,'naeos_identity','USAGE'),false),
		       COALESCE(has_schema_privilege(current_user,'public','CREATE'),false),
		       COALESCE(pg_get_userbyid(n.nspowner),''),
		       current_setting('search_path'),
		       to_regclass('public.goose_db_version') IS NOT NULL,
		       EXISTS (
		         SELECT 1 FROM pg_class c
		         JOIN pg_namespace ns ON ns.oid=c.relnamespace
		         WHERE c.relkind IN ('r','p','v','m')
		           AND ns.nspname IN ('public','naeos_control_plane','naeos_identity')
		           AND (has_table_privilege(current_user,c.oid,'SELECT')
		             OR has_table_privilege(current_user,c.oid,'INSERT')
		             OR has_table_privilege(current_user,c.oid,'UPDATE')
		             OR has_table_privilege(current_user,c.oid,'DELETE'))
		       ),
		       EXISTS (
		         SELECT 1 FROM pg_class c
		         JOIN pg_namespace ns ON ns.oid=c.relnamespace
		         WHERE c.relkind='S'
		           AND ns.nspname IN ('public','naeos_control_plane','naeos_identity')
		           AND (has_sequence_privilege(current_user,c.oid,'USAGE')
		             OR has_sequence_privilege(current_user,c.oid,'SELECT')
		             OR has_sequence_privilege(current_user,c.oid,'UPDATE'))
		       ),
		       EXISTS (
		         SELECT 1 FROM pg_proc p
		         JOIN pg_namespace ns ON ns.oid=p.pronamespace
		         WHERE p.prosecdef
		           AND ns.nspname IN ('public','naeos_control_plane','naeos_identity')
		           AND has_function_privilege(current_user,p.oid,'EXECUTE')
		       )
		FROM pg_roles login
		JOIN pg_roles migrator_group ON migrator_group.rolname='naeos_intelligence_migrator'
		LEFT JOIN pg_namespace n ON n.nspname='naeos_intelligence'
		WHERE login.rolname=session_user
	`).Scan(
		&sessionUser, &currentUser,
		&loginCanLogin, &loginSuper, &loginCreateDB, &loginCreateRole, &loginBypass, &loginInherit,
		&isMember, &membershipCount,
		&groupCanLogin, &groupSuper, &groupCreateDB, &groupCreateRole, &groupBypass, &groupHasParent,
		&schemaExists, &schemaUsage, &schemaCreate, &controlPlaneUsage, &identityUsage, &publicCreate,
		&schemaOwner, &searchPath, &publicLedgerExists, &externalTableAccess, &externalSequenceAccess, &externalSecurityDefinerAccess,
	)
	if err != nil {
		return fmt.Errorf("verify PostgreSQL migrator role contract: %w", err)
	}
	if sessionUser != migratorLoginRole || currentUser != migratorGroupRole ||
		!loginCanLogin || loginSuper || loginCreateDB || loginCreateRole || loginBypass || !loginInherit ||
		!isMember || membershipCount != 1 ||
		groupCanLogin || groupSuper || groupCreateDB || groupCreateRole || groupBypass || groupHasParent ||
		!schemaExists || schemaOwner != migratorGroupRole || !schemaUsage || !schemaCreate ||
		controlPlaneUsage || identityUsage || publicCreate || searchPath != migratorSearchPath ||
		publicLedgerExists || externalTableAccess || externalSequenceAccess || externalSecurityDefinerAccess {
		return errors.New("PostgreSQL migrator role contract rejected: require dedicated non-privileged login, DDL only in naeos_intelligence, no public Goose ledger, and no CRM/Control Plane/Identity privileges")
	}
	return nil
}
