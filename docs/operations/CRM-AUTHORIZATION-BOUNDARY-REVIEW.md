# CRM Authorization Boundary Review — Unified Supabase

**Status:** Source-review and catalog evidence; no production modifications.
**Date:** 2026-11-16
**Scope:** NAEOS CRM authorization model as proposed boundary for the Intelligence control-plane schema.

## Application authorization currently described by repository

CRM README documents:
- JWT bearer-token authentication middleware and OIDC-compatible configuration.
- A development-only `AUTH_DISABLED=true` mode.
- Server-side RBAC for `admin`, `manager`, and `member` roles across routes.
- Consequential operations write durable AuditEvent records.
- Engineering-analysis endpoint is admin-only, read-only, and requires an approved repository allowlist.
- `.env.example` contains `AUTH_SECRET`, `AUTH_DISABLED`, bootstrap credentials, `INTELLIGENCE_API_URL`, repository allowlist and pre-issued service JWT.

The Prisma schema contains `User.supabaseAuthUserId` (nullable unique UUID), `User.roles` (enum array), and `User.status`. These are CRM application identity/RBAC entities. Do not assume these records or the app-level RBAC map are automatically equivalent to Supabase Auth identities, JWT app_metadata, or database role grants.

## Authorization boundary distinction

1. CRM API application auth + RBAC protect requests routed through CRM's API.
2. PostgreSQL RLS protects direct database/API access paths that use Supabase roles.
3. Supabase service role / secret key access bypasses RLS and must stay backend-only.
4. Intelligence service-to-service JWT scopes (`intelligence:read`, `intelligence:write`) authorize connector API calls; they do not grant SQL privileges automatically.
5. The `naeos_control_plane` schema should remain non-exposed and isolated from Prisma-managed `public`.

No policy should be derived solely from a generic CRM role string or `TO authenticated`. Any user-facing policy needs explicit object ownership/tenant scope, action-specific predicate, and tests for cross-user/cross-tenant denial.

## Catalog evidence

- CRM production: 23 tables with RLS enabled and no pg_policies rows (Supabase Security Advisor).
- anon/authenticated have broad table-level privileges on 21 public tables. This is excess privilege surface even when RLS currently denies row access.
- Event trigger `ensure_rls` is enabled on `ddl_command_end`, owned by `postgres`, and invokes `public.rls_auto_enable()`. This auto-enables RLS; it does not generate policies.
- Intelligence routines `append_ledger_event(jsonb,text,text)` and `read_ledger_head()` have SECURITY DEFINER and pinned `search_path=pg_catalog`; execute checks for anon, authenticated, service_role were false. Dedicated Intelligence DB login privilege evidence still must be captured.

## Minimal least-privilege remediation proposal (not applied)

A. Determine CRM API connection role(s) actually used by deployed backend, and identify whether it is Supabase `postgres`, a custom non-BYPASSRLS role, or service role. Inspect deployment configuration names without exposing secret values.
B. For CRM `public`, revoke Data API role table privileges from anon/authenticated where direct client access is not intended; retain only grants proven necessary. Do not alter grants blindly until app connectivity and role ownership are documented.
C. Prefer CRM backend-only access to Prisma-managed public tables. If user direct access is intended, build per-table policies tied to authenticated identity and explicit ownership/tenant columns; this schema presently has no universal tenant/user ownership model, so a blanket `auth.uid()` policy would be incorrect.
D. Create a dedicated Intelligence runtime DB role, non-superuser and NO BYPASSRLS; grant only USAGE on `naeos_control_plane`, narrowly scoped SELECT/INSERT/UPDATE as needed, and EXECUTE on explicitly approved routines. Do not grant `public` schema access by default.
E. Keep migration role distinct from runtime role. Runtime service JWT and SQL DB credentials are separate credentials and separate privilege boundaries.
F. Add negative tests: anon/authenticated cannot read or mutate CRM tables; Intelligence runtime cannot touch CRM CRM-domain tables; only expected ledger routines/tables accessible; unrelated schema access denied; unauthorized ledger writes denied; idempotency and integrity constraints preserved.
G. Validate RLS bypass behavior of privileged roles, SQL function grants including PUBLIC inherited EXECUTE, routine owner, function search_path, and exposure configuration. SECURITY DEFINER requires review of the body and owner privilege boundary, not only path pinning.
H. Staging rehearsals must use synthetic/scrubbed data, migrate schema and seed test rows, execute grants/policies, run application integration tests, inspect advisors, and exercise rollback.

## Release gate

No SQL changes against production or secret/deployment changes are authorized by this review. Proceed only after:
- connection roles and ownership are identified;
- policy matrix is approved by the CRM/Intelligence maintainers;
- backup/restore evidence and staging rehearsal exist;
- security-negative tests pass;
- cutover and rollback steps are signed off.

## Supabase platform inventory cross-check

- Both projects report zero deployed Supabase Edge Functions and only the default `main` development branch in their Supabase branch inventory.
- Supabase migration tracker: staging `naeos.db` has three entries (one historical `new-migration` and two Control Plane migrations); CRM production has zero entries.
- This tracker is distinct from CRM Prisma `_prisma_migrations`, which previously inventoried 12 migration rows. The absence of Supabase-tracker records on CRM does not mean CRM schema has no migrations; Prisma remains the schema lifecycle owner for its `public` tables.
- Cutover plan must explicitly coordinate the Control Plane Supabase migration history and CRM Prisma history without adopting one tool to own the other's schema. Do not apply Intelligence migrations via Prisma.
