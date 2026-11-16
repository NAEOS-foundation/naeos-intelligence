# Intelligence Database Runtime Role and Privilege Matrix

**Status:** Proposed — review required; not an authorization to change database grants  
**Scope:** NAEOS Intelligence application and migration workloads against the database placement selected in ADR-005. ADR-005 states `naeos.db` remains the Intelligence database; `naeos-crm-production` is selected as hosted Supabase Auth IdP. A move of Intelligence persistence into the CRM Supabase PostgreSQL project is not approved by ADR-005 and requires a separate explicit architecture decision.

## 1. Repository evidence

- The HTTP service reads `DATABASE_URL` in `cmd/naeos-intelligence/main.go`. It selects PostgreSQL storage when set and fails startup in production when absent.
- PostgreSQL connection pool is capped at 12 connections, with a 30-minute maximum connection lifetime and 5-minute idle timeout (`internal/postgres/pool.go`).
- The application server does not run schema migrations. The separate `naeos-intelligence-migrate` command reads `DATABASE_URL` and optional `MIGRATIONS_DIR` (default `migrations`), then runs Goose Up.
- The current migration package opens a SQL connection and pings the target but does not enforce a dedicated migration identity itself.
- The IAM decision worksheet still has IAM-03 through IAM-14 open. This document therefore avoids assigning tenant or user-level authorization semantics.

## 2. Live catalog audit evidence (read-only, 2026-11-16)

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
| Intelligence API runtime | `naeos_intelligence_runtime` (LOGIN) | Connect to the target database; use only the Intelligence-owned schema; CRUD on application-owned tables as needed by the implemented API; execute specifically reviewed internal routines | DDL, ownership, role administration, access to CRM `public` tables, broad schema access, arbitrary SECURITY DEFINER execution |
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


## 11. Architecture placement clarification (documentation correction, 2026-11-16)

The earlier scope text in this document implied an approved migration of Intelligence persistence into CRM Supabase. That implication was not supported by ADR-005. ADR-005 explicitly states that `naeos-crm-production` is the hosted identity provider and `naeos.db` remains the Intelligence database; it also states that cross-project database access is not automatically granted by a token issued by the CRM project. This role matrix is now scoped to ADR-005's current placement, not an assumed database consolidation.

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

### Schema placement and migration identity gap

- Both Goose migration files use unqualified object names. The migration command does not set a dedicated schema/search_path; under the normal default PostgreSQL connection search_path this places objects in `public`. The current code therefore does not yet implement the matrix's proposed dedicated Intelligence schema isolation.
- The migration command and API both read `DATABASE_URL`, but are separate executables. The repository does not enforce a distinct migrator credential or a different environment-variable name; separation is currently an operational deployment requirement, not a code-enforced guarantee.
- Consequently, the proposed dedicated-schema and dedicated-migrator model must not be treated as the current deployed state. A grant-ready migration plan needs an accepted schema name, explicit object qualification/search_path strategy, owner model, and a distinct protected migration secret before implementation.

### IAM and authorization boundary

ADR-005 accepts only IAM-01 and IAM-02. IAM-03 through IAM-14 remain open in the authoritative worksheet. In particular, the repository has not yet accepted the canonical mapping from Supabase `iss + sub` to a principal/client, single-tenant versus multi-tenant operating mode, trusted client-to-tenant binding, scope vocabulary, resource ownership, or audit attribution semantics. The fact that current SQL predicates use `client_id` does not by itself prove that the value is a stable, non-spoofable tenant/client identity. Do not convert the present column name into a tenant authorization policy or final table-level grant decision.

## 13. CI and implementation verification status (updated 2026-11-16)

- **Go toolchain security blocker resolved:** PR #75 was merged to `main` and updates the module's Go directive to Go 1.26.11, which addresses the Go 1.26.10 standard-library vulnerability GO-2026-6859.
- **Evidence durability implementation resolved:** PR #76 was merged to `main` at `563562f1fc8dd21fd25ff6096e1df76f6898502f`. Its latest CI run `43825003463` passed module-sum verification, third-party notices, `govulncheck`, secret scanning, `go vet`, unit tests, PostgreSQL 16 integration tests, and build.
- PR #76 also adds transactionally atomic evidence batches, deterministic event identities with replay/conflict handling, stable verification IDs, verification correlation IDs, and fail-closed handling for failed/inconclusive verification outcomes.
- These merged implementation changes do **not** authorize production database changes or establish a grant-ready role matrix. The IAM-03 through IAM-14 decisions remain open; dedicated schema and migrator identity are not yet enforced by the repository; backup/restore evidence and staging rehearsal remain prerequisites.
- Keep this role-matrix PR **Draft** until the outstanding identity/tenant semantics, authorization boundaries, schema/role ownership model, and staging verification prerequisites have been resolved and reviewed. Do not bypass security gates or infer database privileges from column names.

This section records repository and CI evidence only. It does not change database state, runtime configuration, grants, roles, migrations, secrets, or production deployment state.
