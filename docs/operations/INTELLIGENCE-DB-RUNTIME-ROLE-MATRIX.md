# Intelligence Database Runtime Role and Privilege Matrix

**Status:** Proposed — refreshed with read-only staging evidence on 2026-11-22; not grant-ready and not an authorization to change database grants  
**Scope:** NAEOS Intelligence application and migration workloads against the shared staging Supabase project naeos.db (kqmhpsftijskesjdovjm). The project currently hosts CRM staging objects in public, Control Plane objects in naeos_control_plane, and identity/tenant objects in naeos_identity. naeos-crm-production remains the selected hosted Supabase Auth IdP. Production was not accessed or changed. This matrix does not assume the Intelligence Goose persistence tables are already deployed to naeos.db.

## 1. Repository evidence

- The HTTP service reads `DATABASE_URL` in `cmd/naeos-intelligence/main.go`. It selects PostgreSQL storage when set and fails startup in production when absent.
- PostgreSQL connection pool is capped at 12 connections, with a 30-minute maximum connection lifetime and 5-minute idle timeout (`internal/postgres/pool.go`).
- The application server does not run schema migrations. The separate `naeos-intelligence-migrate` command reads `DATABASE_URL` and optional `MIGRATIONS_DIR` (default `migrations`), then runs Goose Up.
- The current migration package opens a SQL connection and pings the target but does not enforce a dedicated migration identity itself.
- Current IAM status is recorded in ADR-0004/ADR-005 and the Phase 5 worksheet: canonical human principal (iss + sub), separate client_id, and human-owner semantics are accepted for the read-only API; ES256 and dedicated naeos_scopes policy are accepted as implementation decisions. Live issuer/JWKS evidence, hook activation, maximum token lifetime, authoritative session revocation, and IAM-14 audit attribution remain gated. This does not establish database runtime identity or grant readiness.

## 2A. Historical catalog audit evidence (read-only, 2026-11-16; superseded by the 2026-11-22 addendum)

### Source project `naeos.db` (`kqmhpsftijskesjdovjm`)

- Source schema `naeos_control_plane` contains six tables: `active_policies`, `approvals`, `grants`, `ledger_events`, `ledger_head`, and `policy_versions`.
- All six tables have RLS enabled; none has FORCE RLS.
- Runtime role currently ACLs: `naeos_control_plane_runtime` has table privileges scoped per table: active_policies/approvals/grants = INSERT, SELECT, UPDATE (ACL notation `arw`); ledger_events = SELECT; policy_versions = INSERT, SELECT. `ledger_head` has no explicit runtime ACL.
- RLS policy inventory: permissive `USING true`/ `WITH CHECK true` for runtime on active_policies, approvals, grants; SELECT policy for ledger_events; INSERT and SELECT policies for policy_versions. No policy exists on ledger_head.
- Four routines are present: `append_ledger_event(jsonb,text,text)` and `read_ledger_head()` are SECURITY DEFINER, owned by postgres, with `search_path=pg_catalog`; EXECUTE ACL is explicit to postgres and `naeos_control_plane_runtime`. Two mutation-rejection trigger routines are SECURITY INVOKER and postgres-only ACL.
- Index inventory includes primary/unique indexes and lookup indexes on approvals decision_id, grants agent_id/policy_id+policy_version, and ledger_events agent/time, decision_id, request_id, event_id. Source advisor flags one informational finding: RLS enabled without policy on `ledger_head`.
- Source has only six schema tables and no view/materialized view/sequence in this schema in the captured inventory.

### CRM production target (`oruykkehukoiipldzcvc`)

- Catalog query found no objects in `naeos_control_plane`, `naeos_identity`, or `intelligence` at audit time.
- CRM application tables remain in Prisma-owned `public`; the Supabase security advisor reports 23 public tables with RLS enabled but no policies. This advisor output is an indicator to review exposure and effective access, not proof that rows are accessible.
- Prior catalog inventory showed broad direct ACLs to `anon` and `authenticated` on most CRM `public` tables. RLS and grants must be reviewed together; do not mass-revoke without application dependency and negative/positive test evidence.
- Security advisor finding remediation reference: https://supabase.com/docs/guides/database/database-linter?lint=0008_rls_enabled_no_policy

**Placement and domain conclusion:** ADR-005 names `naeos.db` as the Intelligence database and `naeos-crm-production` as the hosted Auth IdP. The fact that the CRM project lacks Intelligence schemas is therefore not itself evidence of a deployment defect; it is consistent with the ADR's currently recorded split-project placement. Any proposed unified move of Intelligence persistence/control-plane data into CRM production would conflict with ADR-005's current placement statement and must be captured in a separately approved architecture decision before implementation. `naeos.db` is not approved as unrestricted/general staging. This audit did not inspect or alter data, roles, grants, migrations, or secrets.

## 3. Proposed identities

| Workload | Proposed database identity | Allowed use | Explicitly not allowed |
|---|---|---|---|
| Intelligence API runtime | `naeos_intelligence_runtime` (NOLOGIN group) plus `naeos_intelligence_runtime_login` (LOGIN, INHERIT) | Connect only to `naeos_intelligence` schema; use explicit runtime CRUD grants and reviewed routines | DDL, ownership, role administration, access to CRM `public`, Control Plane or Identity schemas, broad schema access, arbitrary SECURITY DEFINER execution |
| Intelligence schema migration job | `naeos_intelligence_migrator` (LOGIN, short-lived/controlled) | Apply approved Goose migrations to Intelligence-owned objects; own or receive only the DDL privileges required for that schema | CRM application-data access, public API exposure, long-running runtime use, superuser or project-wide admin |
| Supabase project administration | Existing trusted administrator / platform role | Bootstrap identities, backup/restore, reviewed grant changes, platform administration | Application connection use; routine deploy credential use |

Names are proposed only. Confirm the target schema name, database role capabilities, provider constraints, and owner model before creating identities.

## 4. Privilege matrix (least-privilege target)

| Privilege | Runtime | Migrator | Admin |
|---|---:|---:|---:|
| CONNECT to target database | Yes | Yes | Yes |
| USAGE on Intelligence schema | Yes | Yes | Yes |
| SELECT / INSERT / UPDATE / DELETE on runtime-required Intelligence tables | Only per reviewed table/action inventory | Usually yes for migration verification; not assumed by default | Yes |
| USAGE / SELECT on sequences, if any | Only required sequences | As migration needs | Yes |
| CREATE on Intelligence schema | No | Yes, only during migration workflow or through controlled owner model | Yes |
| CREATE on `public` / CRM schema | No | No | Controlled |
| ALTER / DROP Intelligence objects | No | Yes, for approved migrations | Yes |
| Ownership of CRM `public` objects | No | No | Existing owner only |
| Execute reviewed routines | Explicit allowlist only | Migration-required only | Yes |
| Execute SECURITY DEFINER routines | Deny by default; grant only after body, owner, ACL, and call-path review | Deny by default | Controlled |
| BYPASSRLS / superuser / role creation | No | No | Platform-controlled only |

This matrix is a target model, not a grant script. Table privileges must be refined after reading the actual SQL migration definitions and runtime query paths.

## 5. Isolation and namespace controls

1. Intelligence objects must be contained in a dedicated schema distinct from CRM Prisma-owned `public` objects.
2. Runtime role must not inherit broad privileges on `public`, nor receive privileges through `PUBLIC` that defeat the intended boundary.
3. Verify effective privileges, including role memberships, inherited privileges, default privileges, schema ACLs, object ACLs, and `PUBLIC` EXECUTE.
4. Pin a safe `search_path` for each application connection; avoid relying on mutable schema resolution. Explicitly qualify Intelligence object references where practical.
5. Keep CRM application authorization (JWT, CRM RBAC, API scopes) distinct from PostgreSQL grants. Database role separation is an independent defense layer.
6. Never put the migration connection string in the API runtime deployment. Keep it in a restricted migration workflow secret.
7. Do not expose the Intelligence schema through Supabase Data API unless an explicit use case, RLS policy design, and review approve it.

## 6. SECURITY DEFINER routine review gate

For each existing or proposed SECURITY DEFINER routine, record:
- exact schema/name/signature and owner;
- routine body and data touched;
- pinned `search_path`;
- ACL including PUBLIC, anon, authenticated, service_role, runtime, and migrator;
- necessity of SECURITY DEFINER versus SECURITY INVOKER;
- authorization checks in the routine and its callers;
- expected execution identity and negative tests.

No blanket EXECUTE grants should be added. Do not revoke or alter production ACLs until dependency evidence and staging tests establish the intended behavior.

## 7. Migration execution controls

- Migration artifact is versioned, reviewed, and tied to a commit SHA.
- Backup and restore verification plus staging rehearsal are prerequisites.
- Apply migrations using the restricted migrator identity, never the runtime identity.
- Capture migration version before/after, command output, schema diff, advisors, smoke tests, and rollback/forward-fix decision.
- Fail closed if target project, database, schema, role, or migration version is ambiguous.
- Production cutover requires explicit approval after staging acceptance.

## 8. Required verification checklist

- [ ] Read every Goose migration and map CRUD/DDL to actual runtime packages.
- [ ] Confirm target connection endpoint mode (direct/session pooler/transaction pooler) and supported role behavior.
- [ ] Confirm roles, memberships, ownership, ACLs, default ACLs, and PUBLIC grants from catalog queries.
- [ ] Confirm Supabase Data API exposure configuration and schema exposure.
- [ ] Run positive and negative privilege tests in staging: runtime can perform expected operations; cannot DDL, access CRM data, or execute unapproved routines.
- [ ] Run migration identity tests separately, including rollback behavior in staging.
- [ ] Run security/performance advisors and capture findings.
- [ ] Establish backup/restore evidence and staging rehearsal.
- [ ] Obtain explicit production change approval.

## 9. Current non-actions

This document does not create roles, change grants, modify RLS, run migrations, move data, configure Supabase Auth, or change production deployment secrets.


## 10. Goose migration and runtime query-path reconciliation (read-only, 2026-11-16)

### Migration-defined objects

The repository's current Goose Up migrations create objects in the default schema (unqualified names; normally `public` under the migration connection's `search_path`):

| Migration | Objects | Constraints / behavior |
|---|---|---|
| `00001_request_persistence.sql` | `intelligence_requests`, `request_transitions`, `request_attempts`, index set, `prevent_request_transition_mutation()`, trigger | Requests are mutable lifecycle records; transitions have UPDATE/DELETE rejection trigger; attempts support stateful writes |
| `00002_governance_approval_audit.sql` | `approval_decisions`, `governance_audit_events`, indexes, `prevent_governance_audit_mutation()`, `prevent_approval_decision_mutation()`, triggers | Approval decisions and audit events are append-only via UPDATE/DELETE rejection triggers |

The migrations do **not** create the source project's six `naeos_control_plane` tables. They also do not create a dedicated schema, RLS policies, table grants, runtime/migrator roles, or Supabase Data API exposure configuration. Goose migration source should therefore not be treated as a full deployment/IAM specification.

### Runtime CRUD evidenced from code

| Table | Runtime package / path | Observed operations |
|---|---|---|
| `intelligence_requests` | `internal/postgres/store.go`; `internal/postgres/approval_store.go` | INSERT, SELECT, UPDATE (cancel/status transition) |
| `request_transitions` | `internal/postgres/store.go`; `internal/postgres/approval_store.go` | INSERT only in reviewed call paths |
| `request_attempts` | `internal/postgres/store.go` | SELECT, INSERT, UPDATE |
| `approval_decisions` | `internal/postgres/approval_store.go` | SELECT, INSERT |
| `governance_audit_events` | `internal/approval/store.go`, `internal/evidence/store.go` | INSERT, SELECT |

This is a code-path-derived CRUD inventory, not yet a grant-ready matrix: call-path reachability, deployment identity, tenant isolation enforcement, effective grants, and any uninspected code paths remain to be tested. No DELETE operation was identified in the reviewed runtime persistence files. Trigger functions are invoked by triggers; no direct application EXECUTE grant is evidenced as required by these call paths.

### Reconciliation finding / blocker

The live source schema (`naeos_control_plane`) holds policy/grant/approval/ledger control-plane records. The Goose migrations define request lifecycle, attempt, approval-decision, and governance-audit persistence used by the reviewed Intelligence runtime. Repository search did not find direct references to the source schema's table names in indexed code paths; this is not proof that every runtime module has been exhaustively checked. These may be distinct bounded contexts rather than duplicate schemas to merge. Preserve their separation until a data lineage/ownership map and explicit ADR decide whether both are required, how events/records relate, and which database hosts each. ADR-005 currently assigns Intelligence database placement to `naeos.db`; moving these objects to CRM production would require a decision amendment. Do not generate or apply grants, migrations, or data mapping from name similarity alone.

### Verification boundary

Evidence reviewed: Goose files `migrations/00001_request_persistence.sql` and `00002_governance_approval_audit.sql`; runtime persistence files `internal/postgres/store.go`, `internal/postgres/approval_store.go`, and `internal/evidence/store.go`. This section is documentation-only. No SQL writes, DDL, role/grant changes, migrations, data movement, secrets changes, or production deployment actions were performed.


## 11. Architecture placement clarification (documentation correction, 2026-11-16; refined by 2026-11-22 staging evidence)

The earlier scope text in this document implied an approved migration of Intelligence persistence into CRM Supabase. That implication was not supported by ADR-005. ADR-005 explicitly selects naeos-crm-production as the hosted identity provider and naeos.db as the Intelligence staging database; cross-project database access is not automatically granted by a token issued by the CRM project. The 2026-11-22 live catalog audit confirms that naeos.db is shared with CRM staging and the Control Plane/Identity schemas, rather than being an exclusive Intelligence-only project. This role matrix is now scoped to ADR-005's current placement, not an assumed database consolidation.

**Architecture gate:** a unified-database placement change must be separately proposed, assessed (including cross-project trust, ownership, RLS/Data API exposure, role boundaries, backup/restore, and migration rehearsal), and accepted by the maintainer before its implementation PR is created or production change is planned. Existing PR #73's unified migration plan remains a proposal/draft and does not supersede ADR-005.

This clarification is a documentation-only change; no database or deployment state was modified.

## 12. Expanded runtime call-path reconciliation (read-only, 2026-11-16)

### Active HTTP request and evidence paths

- `cmd/naeos-intelligence/main.go` creates one PostgreSQL pool with `postgres.OpenPool`, passes that pool to `postgres.New` for the HTTP request store, and passes the same pool to `evidence.New` for the orchestrator evidence store. The current process therefore does not use a separate database identity for request persistence and evidence persistence.
- Protected HTTP handlers obtain `principal.ClientID` from authenticated request context and pass it into request create/get/cancel/approval operations. The reviewed `internal/postgres/store.go` and `internal/postgres/approval_store.go` queries scope request and approval-decision reads/writes by `client_id`.
- The active orchestrator path calls `evidence.Store.RecordEvent`; this inserts into `governance_audit_events` with the caller-supplied client ID. `GetEvents` also filters by client ID and request ID. For the currently wired runtime path, the role therefore needs the reviewed INSERT path for evidence events; SELECT is required only if/when the corresponding read path is exposed or called.
- The active runtime CRUD inventory in section 10 should be interpreted together with this wiring: `governance_audit_events` INSERT is active through the orchestrator, while its SELECT path exists in the store but is not shown as an HTTP route in the reviewed handler.

### Dormant or currently unwired persistence paths

- Repository-wide code search found no current import/instantiation of `internal/approval.Store` from the running API composition. Its implementation nevertheless contains request-ID-only queries against `approval_decisions`, a global `Decisions` list, and inserts with `client_id='system'`. These methods are not included in the active runtime privilege grant model. Before wiring them, redesign them to carry a trusted client/tenant context and apply consistent ownership predicates; add cross-client negative tests. Do not grant global approval-table access on the assumption that this dormant package is active.
- `CreateAttempt`, `StartAttempt`, and `FinishAttempt` are present in `internal/postgres/store.go`, but repository search found no active API/orchestrator call sites beyond the implementation and integration-test references for attempt creation. Treat attempt-table privileges as not yet justified by the current exposed request path; reassess when a worker/attempt lifecycle is wired.
- **Resolved by PR #76 (merged 2026-11-16):** the active orchestrator now requires a durable evidence recorder, persists the classified/planned/verification/completion event batch atomically, and returns an error if persistence fails. A failed or inconclusive verification is durably recorded without an `orchestration.completed` event. The in-memory completion marker is written only after the evidence transaction commits. This closes the evidence-durability implementation finding; it does not resolve the outstanding IAM, schema-isolation, migrator-identity, backup/restore, or staging gates.

### Schema placement and migration identity gap (repository state inspected 2026-11-16; live staging confirmation in 2026-11-22 addendum)

- Both Goose migration files use unqualified object names. The migration command does not set a dedicated schema/search_path; under the normal default PostgreSQL connection search_path this places objects in `public`. The current code therefore does not yet implement the matrix's proposed dedicated Intelligence schema isolation.
- The migration command and API both read `DATABASE_URL`, but are separate executables. The repository does not enforce a distinct migrator credential or a different environment-variable name; separation is currently an operational deployment requirement, not a code-enforced guarantee.
- Consequently, the proposed dedicated-schema and dedicated-migrator model must not be treated as the current deployed state. A grant-ready migration plan needs an accepted schema name, explicit object qualification/search_path strategy, owner model, and a distinct protected migration secret before implementation.

### IAM and authorization boundary (historical status 2026-11-16; see current status in addendum)

ADR-005 accepts only IAM-01 and IAM-02. IAM-03 through IAM-14 remain open in the authoritative worksheet. In particular, the repository has not yet accepted the canonical mapping from Supabase `iss + sub` to a principal/client, single-tenant versus multi-tenant operating mode, trusted client-to-tenant binding, scope vocabulary, resource ownership, or audit attribution semantics. The fact that current SQL predicates use `client_id` does not by itself prove that the value is a stable, non-spoofable tenant/client identity. Do not convert the present column name into a tenant authorization policy or final table-level grant decision.

## 13. Historical CI and implementation verification status (2026-11-16; superseded by the 2026-11-22 addendum)

- **Go toolchain security blocker resolved:** PR #75 was merged to `main` and updates the module's Go directive to Go 1.26.11, which addresses the Go 1.26.10 standard-library vulnerability GO-2026-6859.
- **Evidence durability implementation resolved:** PR #76 was merged to `main` at `563562f1fc8dd21fd25ff6096e1df76f6898502f`. Its latest CI run `43825003463` passed module-sum verification, third-party notices, `govulncheck`, secret scanning, `go vet`, unit tests, PostgreSQL 16 integration tests, and build.
- PR #76 also adds transactionally atomic evidence batches, deterministic event identities with replay/conflict handling, stable verification IDs, verification correlation IDs, and fail-closed handling for failed/inconclusive verification outcomes.
- These merged implementation changes do **not** authorize production database changes or establish a grant-ready role matrix. The IAM-03 through IAM-14 decisions remain open; dedicated schema and migrator identity are not yet enforced by the repository; backup/restore evidence and staging rehearsal remain prerequisites.
- Keep this role-matrix PR **Draft** until the outstanding identity/tenant semantics, authorization boundaries, schema/role ownership model, and staging verification prerequisites have been resolved and reviewed. Do not bypass security gates or infer database privileges from column names.

This section records repository and CI evidence only. It does not change database state, runtime configuration, grants, roles, migrations, secrets, or production deployment state.


---

# 2026-11-22 live staging audit addendum

**Audit mode:** read-only catalog inspection of the naeos.db staging project only. No SQL writes, DDL, grants, role provisioning, migration, secret read, production access, or deployment occurred.

## A. Project and schema inventory

- Supabase project: naeos.db (kqmhpsftijskesjdovjm), ACTIVE_HEALTHY, PostgreSQL 17.6.1, region ap-northeast-1.
- User schemas inspected: public, naeos_control_plane, naeos_identity.
- public: 22 CRM staging tables.
- naeos_control_plane: 6 tables — active_policies, approvals, grants, ledger_events, ledger_head, policy_versions.
- naeos_identity: 7 tables — audit_events, memberships, organizations, role_assignments, role_capabilities, role_definitions, workspaces.
- All 35 tables are owned by postgres, have RLS enabled, and have FORCE RLS disabled. No views, materialized views, or sequences were found in these three schemas.
- Supabase migration history reports 20 applied migrations, latest 20261122000100 atomic_execution_reconciliation.

## B. Public CRM schema — effective access

- All 22 public CRM tables have RLS enabled but no RLS policies.
- Catalog ACLs and effective privilege checks show anon and authenticated have zero SELECT/INSERT/UPDATE/DELETE privileges on these tables. The public schema itself has USAGE for these roles, but table privileges are absent.
- service_role has SELECT/INSERT/UPDATE/DELETE on all 22 tables and has BYPASSRLS. It is a NOLOGIN PostgREST role reached through the authenticator role; it must remain server-side and must not be used as an application database login.
- Current CRM tables are owned by postgres; their table ACLs grant full access to postgres and service_role, not anon/authenticated.
- Default privileges for objects created by supabase_admin in public include broad grants to anon, authenticated, and service_role for tables, sequences, and functions. This is a future-object exposure consideration; it is not evidence that the current 22 CRM tables are directly accessible to browser roles. Do not mass-revoke platform defaults without a dependency review.
- Security advisor reports 22 public rls_enabled_no_policy findings. Interpret this alongside the zero table privileges for anon and authenticated; the finding is a review signal, not proof of public row access.

## C. Control Plane schema — role and policy findings

- naeos_control_plane_runtime exists as NOLOGIN, INHERIT, NOBYPASSRLS, non-superuser, non-creator.
- It has effective table privileges across the six tables: 5 SELECT, 4 INSERT, 3 UPDATE, and 0 DELETE. ledger_head has no direct runtime table privilege and no RLS policy; read_ledger_head() is the intended read path.
- Six RLS policies target naeos_control_plane_runtime. Several are permissive USING (true) / WITH CHECK (true); table ACLs therefore define the role's effective write surface.
- Runtime currently can INSERT policy snapshots and UPDATE the active policy pointer, grants, and approvals. This is policy-authoring/administration authority, not merely policy evaluation. It is a blocker for a production runtime identity until a separate policy-administration role or an explicit, narrowly scoped threat-model exception is accepted. See the NAEOS Core architecture gate in production-authorization-execution-separation.md (Issue #848).
- append_ledger_event(jsonb,text,text), read_ledger_head(), and reconcile_execution(text,jsonb) are SECURITY DEFINER with search_path=pg_catalog; EXECUTE ACL is explicit to postgres and naeos_control_plane_runtime. Trigger rejection routines are SECURITY INVOKER and postgres-only.

## D. Identity schema — role and policy findings

- naeos_identity_runtime and naeos_identity_admin exist as NOLOGIN, INHERIT, NOBYPASSRLS, non-superuser, non-creator roles.
- naeos_identity_admin is a member of naeos_identity_runtime; it inherits runtime read privileges.
- Runtime has SELECT on all seven identity tables and INSERT only on audit_events; it has no UPDATE/DELETE privileges.
- Admin has SELECT/INSERT on all seven tables and UPDATE/DELETE on six; audit-event mutation is guarded by immutable triggers.
- Sixteen RLS policies scope runtime access through transaction-local settings (naeos.issuer, naeos.subject, naeos.organization_id, naeos.workspace_id) and active membership/capability checks. These GUC values are not an independent authentication boundary: only a trusted backend may set them after token verification and membership resolution.
- is_active_tenant(uuid,uuid) is SECURITY DEFINER with search_path=pg_catalog, naeos_identity; EXECUTE is explicitly granted to postgres, runtime, and admin roles. Runtime and admin have USAGE but no CREATE on the schema.
- Eight performance advisor findings report per-row re-evaluation of current_setting() in identity RLS policies; five unindexed foreign keys and 77 unused-index notices are also reported. These are performance review items, not authorization proof.

## E. Runtime identity and migration evidence — BLOCKED

- The staging catalog contains no roles named naeos_intelligence_runtime, naeos_intelligence_runtime_login, naeos_intelligence_migrator, or naeos_intelligence_migrator_login.
- The Control Plane/Identity login roles described in migration comments (for example naeos_control_plane_login, naeos_identity_runtime_login, and naeos_identity_admin_login) are also absent. The group roles are NOLOGIN; role membership shows postgres as a member, and naeos_identity_admin as a member of naeos_identity_runtime. The authenticator login is a member of anon, authenticated, and service_role, not the custom NAEOS runtime roles.
- pg_stat_activity snapshot showed PostgREST (authenticator), PgBouncer, Supabase management, pg_net, and Supabase platform/admin connections; no NAEOS Intelligence or Control Plane runtime session was present at audit time.
- naeos.db does not contain intelligence_requests, request_transitions, request_attempts, approval_decisions, governance_audit_events, goose_db_version, or public.schema_migrations. Supabase migration history contains CRM, Control Plane, and Identity migrations, not the repository's Goose persistence migrations.
- Repository code evidence: cmd/naeos-intelligence/main.go reads DATABASE_URL; cmd/naeos-intelligence-migrate/main.go also reads DATABASE_URL. internal/postgres/pool.go sets pool sizing/lifetime only; it does not set role or search_path. The migration command does not enforce a distinct migrator identity. Goose migrations 00001 and 00002 use unqualified object names, which resolve through the connection's default search_path (normally public).
- The actual deployment DATABASE_URL was not read. No application database connection was observed in the staging snapshot. Therefore it is BLOCKED to claim that NAEOS Intelligence currently uses the naeos.db project or its NOLOGIN runtime roles. If the same DSN is used for runtime and migration, the separation is not enforced by code.

## F. Security advisor summary

- Security: 23 rls_enabled_no_policy findings — 22 public CRM tables plus naeos_control_plane.ledger_head; one warning reports Supabase Auth leaked-password protection disabled.
- Performance: 5 unindexed foreign keys, 8 RLS initplan warnings, and 77 unused-index notices.
- No advisor remediation was applied. Unused-index findings are not grounds for index deletion in a low-traffic staging database without workload evidence.

## G. Recommended target boundary (proposal; not implemented)

1. Keep the shared staging project, but isolate domain data by schema: CRM staging remains in public; Control Plane remains in naeos_control_plane; Identity remains in naeos_identity; Intelligence request/attempt/approval/evidence persistence moves only after approval into a dedicated naeos_intelligence schema.
2. Use a NOLOGIN group role naeos_intelligence_runtime and a separate LOGIN role naeos_intelligence_runtime_login with membership only in that group. Runtime privileges must match active call paths; no DDL, no access to CRM public tables, no Control Plane policy administration, and no BYPASSRLS.
3. Use a separate NOLOGIN group role naeos_intelligence_migrator and LOGIN role naeos_intelligence_migrator_login, with schema-scoped DDL only. Require a separate MIGRATOR_DATABASE_URL; never reuse the runtime DSN.
4. Schema-qualify Goose migrations and runtime SQL to naeos_intelligence; configure a safe connection search_path (pg_catalog, naeos_intelligence) as defense in depth. Ensure Goose's own migration ledger is also isolated from CRM public.
5. Add startup checks proving session_user is the dedicated runtime login, it is a member of only the expected runtime group, and it is not superuser/CREATEDB/CREATEROLE/BYPASSRLS. Add equivalent migration-job checks for the migrator login.
6. Keep service_role and browser roles out of the custom schemas. Do not create or alter any role/grant until the migration plan, secret-management process, and staging positive/negative tests are reviewed.
7. Split Control Plane policy administration from evaluation. The current naeos_control_plane_runtime grants are too broad for a pure runtime identity; separate policy-authoring and approval administration from the runtime service before staging activation.
8. Provision the Identity runtime/admin LOGIN roles only through a secret-managed, separately reviewed workflow; the current migrations intentionally create only NOLOGIN groups.

## H. Acceptance evidence still required

- [ ] Explicitly confirm whether naeos.db is the target for Intelligence persistence or whether Intelligence remains on a separate database; do not infer from project naming.
- [ ] Decide and approve naeos_intelligence schema placement and ownership model.
- [ ] Read every Goose migration and map active runtime CRUD to table-level grants; exclude dormant approval/attempt paths until wired.
- [ ] Add schema-qualified migration and query tests; assert no Intelligence objects are created in public.
- [ ] Provision staging runtime and migrator LOGIN roles via a secure secret-managed process; no password in repository.
- [ ] Verify runtime and migrator effective privileges, role memberships, default ACLs, schema ACLs, and search_path in staging.
- [ ] Run positive/negative database privilege tests: runtime cannot DDL, cannot access CRM/Control Plane/Identity schemas, cannot change policy/grant/approval records, and cannot execute unapproved SECURITY DEFINER routines.
- [ ] Verify Control Plane policy-administration split and Identity runtime/admin connectivity separately.
- [ ] Confirm Supabase Data API schema exposure settings and effective table grants.
- [ ] Run advisors after approved DDL, then stage backup/restore rehearsal and smoke tests.
- [ ] Obtain explicit approval before any production change.

**Disposition:** database isolation is NOT READY. The catalog contains useful NOLOGIN role groups and RLS policies, but the required application login roles, runtime connection path, Goose schema isolation, and migrator separation are not evidenced. No database mutation or production operation is authorized by this document.
