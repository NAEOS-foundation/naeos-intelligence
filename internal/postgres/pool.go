package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const runtimeGroupRole = "naeos_intelligence_runtime"
const runtimeLoginRole = "naeos_intelligence_runtime_login"
const migratorGroupRole = "naeos_intelligence_migrator"
const migratorLoginRole = "naeos_intelligence_migrator_login"
const runtimeSearchPath = "pg_catalog, naeos_intelligence"
const migratorSearchPath = "naeos_intelligence, pg_catalog"

// OpenPool creates a bounded runtime pool. It refuses privileged, shared, or
// incorrectly scoped database identities before the application serves traffic.
func OpenPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL runtime URL: %w", err)
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = make(map[string]string)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = runtimeSearchPath
	cfg.ConnConfig.RuntimeParams["role"] = runtimeGroupRole
	cfg.MaxConns = 12
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL runtime pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL runtime pool: %w", err)
	}
	if err := verifyRuntimeRole(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func verifyRuntimeRole(ctx context.Context, pool *pgxpool.Pool) error {
	var (
		sessionUser, currentUser, schemaOwner, searchPath string
		loginCanLogin, loginSuper, loginCreateDB, loginCreateRole, loginBypass, loginInherit bool
		isMember, groupCanLogin, groupSuper, groupCreateDB, groupCreateRole, groupBypass, groupHasParent bool
		migratorCanLogin, migratorSuper, migratorCreateDB, migratorCreateRole, migratorBypass, migratorHasParent bool
		schemaExists, schemaUsage, schemaCreate, controlPlaneUsage, identityUsage bool
		requestsSelect, requestsInsert, requestsTableUpdate, requestsDelete bool
		requestsStatusUpdate, requestsUpdatedAtUpdate, requestsOwnerUpdate, requestsPayloadUpdate, requestsIdempotencyUpdate, requestsDigestUpdate bool
		transitionsSelect, transitionsInsert, transitionsUpdate, transitionsDelete bool
		eventsSelect, eventsInsert, eventsUpdate, eventsDelete bool
		attemptsAny, approvalsAny, gooseAny, externalTableAccess, externalSequenceAccess, externalSecurityDefinerAccess, publicCreate bool
		membershipCount int
	)
	err := pool.QueryRow(ctx, `
		SELECT session_user::text, current_user::text,
		       login.rolcanlogin, login.rolsuper, login.rolcreatedb, login.rolcreaterole, login.rolbypassrls, login.rolinherit,
		       pg_has_role(session_user, 'naeos_intelligence_runtime', 'MEMBER'),
		       (SELECT count(*) FROM pg_auth_members WHERE member=login.oid),
		       runtime_group.rolcanlogin, runtime_group.rolsuper, runtime_group.rolcreatedb,
		       runtime_group.rolcreaterole, runtime_group.rolbypassrls,
		       EXISTS (SELECT 1 FROM pg_auth_members WHERE member=runtime_group.oid),
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
		       has_table_privilege(current_user,'naeos_intelligence.intelligence_requests','SELECT'),
		       has_table_privilege(current_user,'naeos_intelligence.intelligence_requests','INSERT'),
		       has_table_privilege(current_user,'naeos_intelligence.intelligence_requests','UPDATE'),
		       has_column_privilege(current_user,'naeos_intelligence.intelligence_requests','status','UPDATE'),
		       has_column_privilege(current_user,'naeos_intelligence.intelligence_requests','updated_at','UPDATE'),
		       has_column_privilege(current_user,'naeos_intelligence.intelligence_requests','client_id','UPDATE'),
		       has_column_privilege(current_user,'naeos_intelligence.intelligence_requests','payload','UPDATE'),
		       has_column_privilege(current_user,'naeos_intelligence.intelligence_requests','idempotency_key','UPDATE'),
		       has_column_privilege(current_user,'naeos_intelligence.intelligence_requests','payload_digest','UPDATE'),
		       has_table_privilege(current_user,'naeos_intelligence.intelligence_requests','DELETE'),
		       has_table_privilege(current_user,'naeos_intelligence.request_transitions','SELECT'),
		       has_table_privilege(current_user,'naeos_intelligence.request_transitions','INSERT'),
		       has_table_privilege(current_user,'naeos_intelligence.request_transitions','UPDATE'),
		       has_table_privilege(current_user,'naeos_intelligence.request_transitions','DELETE'),
		       has_table_privilege(current_user,'naeos_intelligence.governance_audit_events','SELECT'),
		       has_table_privilege(current_user,'naeos_intelligence.governance_audit_events','INSERT'),
		       has_table_privilege(current_user,'naeos_intelligence.governance_audit_events','UPDATE'),
		       has_table_privilege(current_user,'naeos_intelligence.governance_audit_events','DELETE'),
		       (has_table_privilege(current_user,'naeos_intelligence.request_attempts','SELECT')
		        OR has_table_privilege(current_user,'naeos_intelligence.request_attempts','INSERT')
		        OR has_table_privilege(current_user,'naeos_intelligence.request_attempts','UPDATE')
		        OR has_table_privilege(current_user,'naeos_intelligence.request_attempts','DELETE')),
		       (has_table_privilege(current_user,'naeos_intelligence.approval_decisions','SELECT')
		        OR has_table_privilege(current_user,'naeos_intelligence.approval_decisions','INSERT')
		        OR has_table_privilege(current_user,'naeos_intelligence.approval_decisions','UPDATE')
		        OR has_table_privilege(current_user,'naeos_intelligence.approval_decisions','DELETE')),
		       (has_table_privilege(current_user,'naeos_intelligence.goose_db_version','SELECT')
		        OR has_table_privilege(current_user,'naeos_intelligence.goose_db_version','INSERT')
		        OR has_table_privilege(current_user,'naeos_intelligence.goose_db_version','UPDATE')
		        OR has_table_privilege(current_user,'naeos_intelligence.goose_db_version','DELETE')),
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
		JOIN pg_roles runtime_group ON runtime_group.rolname='naeos_intelligence_runtime'
		JOIN pg_roles migrator_group ON migrator_group.rolname='naeos_intelligence_migrator'
		LEFT JOIN pg_namespace n ON n.nspname='naeos_intelligence'
		WHERE login.rolname=session_user
	`).Scan(
		&sessionUser, &currentUser,
		&loginCanLogin, &loginSuper, &loginCreateDB, &loginCreateRole, &loginBypass, &loginInherit,
		&isMember, &membershipCount,
		&groupCanLogin, &groupSuper, &groupCreateDB, &groupCreateRole, &groupBypass, &groupHasParent,
		&migratorCanLogin, &migratorSuper, &migratorCreateDB, &migratorCreateRole, &migratorBypass, &migratorHasParent,
		&schemaExists, &schemaUsage, &schemaCreate, &controlPlaneUsage, &identityUsage, &publicCreate,
		&schemaOwner, &searchPath,
		&requestsSelect, &requestsInsert, &requestsTableUpdate,
		&requestsStatusUpdate, &requestsUpdatedAtUpdate, &requestsOwnerUpdate, &requestsPayloadUpdate, &requestsIdempotencyUpdate, &requestsDigestUpdate,
		&requestsDelete,
		&transitionsSelect, &transitionsInsert, &transitionsUpdate, &transitionsDelete,
		&eventsSelect, &eventsInsert, &eventsUpdate, &eventsDelete,
		&attemptsAny, &approvalsAny, &gooseAny, &externalTableAccess, &externalSequenceAccess, &externalSecurityDefinerAccess,
	)
	if err != nil {
		return fmt.Errorf("verify PostgreSQL runtime role contract: %w", err)
	}
	if sessionUser != runtimeLoginRole || currentUser != runtimeGroupRole ||
		!loginCanLogin || loginSuper || loginCreateDB || loginCreateRole || loginBypass || !loginInherit ||
		!isMember || membershipCount != 1 ||
		groupCanLogin || groupSuper || groupCreateDB || groupCreateRole || groupBypass || groupHasParent ||
		migratorCanLogin || migratorSuper || migratorCreateDB || migratorCreateRole || migratorBypass || migratorHasParent ||
		!schemaExists || schemaOwner != migratorGroupRole || !schemaUsage || schemaCreate ||
		controlPlaneUsage || identityUsage || publicCreate || searchPath != runtimeSearchPath ||
		!requestsSelect || !requestsInsert || requestsTableUpdate || requestsDelete ||
		!requestsStatusUpdate || !requestsUpdatedAtUpdate || requestsOwnerUpdate || requestsPayloadUpdate || requestsIdempotencyUpdate || requestsDigestUpdate ||
		transitionsSelect || !transitionsInsert || transitionsUpdate || transitionsDelete ||
		!eventsSelect || !eventsInsert || eventsUpdate || eventsDelete ||
		attemptsAny || approvalsAny || gooseAny || externalTableAccess || externalSequenceAccess || externalSecurityDefinerAccess {
		return errors.New("PostgreSQL runtime role contract rejected: require dedicated non-privileged login, naeos_intelligence-only access, owner-scoped CRUD, append-only event writes, and no worker/approval privileges")
	}
	return nil
}
