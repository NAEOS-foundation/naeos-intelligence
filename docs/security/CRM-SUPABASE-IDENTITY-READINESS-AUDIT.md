# CRM Supabase Identity Readiness — Read-Only Audit

**Date:** 2026-11-11  
**Project:** `naeos-crm-production`  
**Project ref:** `oruykkehukoiipldzcvc`  
**Region:** `ap-southeast-1` (Singapore)  
**Status:** Findings recorded; remediation not authorized or applied

## 1. Scope and method

This is a read-only inspection of Supabase project metadata, public-schema table metadata, PostgreSQL RLS policy metadata, function definition/ACL metadata, and Supabase security advisor output. No DDL, DCL, row data, key rotation, Auth settings, or production configuration was changed.

## 2. Identity-provider readiness

- Project status returned by Supabase: `ACTIVE_HEALTHY`.
- The project is the maintainer-selected hosted Supabase Auth project for NAEOS.
- The active JWT signing algorithm and key were **not** verifiable through the available project metadata connector.
- The public JWKS endpoint could not be fetched from the current inspection environment. This is an inspection-path limitation, not evidence that the endpoint is down.
- No real session access JWT was inspected. Therefore exact `iss`, `aud`, `alg`, `kid`, `sub`, `exp`, and optional `nbf` values remain unverified.

**Gate:** Do not set `NAEOS_AUTH_ISSUER`, `NAEOS_AUTH_AUDIENCE`, or `NAEOS_AUTH_JWKS_URL` in production until the active RS256 key, live JWKS, and real token claims have been independently confirmed.

Candidate endpoints for this project:

- Issuer: `https://oruykkehukoiipldzcvc.supabase.co/auth/v1`
- JWKS: `https://oruykkehukoiipldzcvc.supabase.co/auth/v1/.well-known/jwks.json`
- Candidate audience: `authenticated` (must be checked against a real access token)

## 3. RLS inventory

A catalog query returned **23 public tables; all 23 have RLS enabled and zero policies**. PostgreSQL privilege checks also showed `anon` and `authenticated` have table-level SELECT grants on the inspected public tables (including representative `User` and `Lead` tables).

Tables with RLS enabled and no policies:

- `Activity`
- `AuditEvent`
- `Campaign`
- `CampaignStep`
- `Community`
- `Company`
- `Contact`
- `Contributor`
- `EvidenceReceipt`
- `FollowUp`
- `GoGateRequest`
- `Investor`
- `Lead`
- `Opportunity`
- `OutboxEvent`
- `Partner`
- `PipelineStage`
- `PolicyDecision`
- `PolicyRule`
- `Task`
- `UseCase`
- `User`
- `_prisma_migrations`

With RLS enabled and no applicable policy, ordinary `anon`/`authenticated` table access is expected to be denied by RLS despite the SQL grants. This is a default-deny posture, not evidence of public row disclosure. It may, however, break CRM features if the browser uses Supabase Data API directly. A trusted backend using a privileged database role can bypass RLS, so its access boundary must be verified separately.

**Required design work before creating policies:**

1. Establish whether CRM accesses these tables through Prisma/backend-only connections, Supabase Data API, or both.
2. Classify tables as browser-exposed application data, backend-only operational data, audit/evidence records, or internal migration metadata.
3. Define the authenticated subject-to-CRM-user mapping and role/ownership semantics.
4. Design least-privilege SELECT/INSERT/UPDATE/DELETE policies per table and operation; do not copy a blanket `auth.uid()` policy because the CRM schema uses text user IDs and the mapping is not yet verified.
5. Keep `_prisma_migrations`, outbox, policy, execution-gate, and evidence tables backend-only unless a reviewed requirement proves otherwise.
6. Test both permitted and denied access using real anon/authenticated JWT contexts and the service role only in trusted server-side tests.

## 4. SECURITY DEFINER event-trigger function

Supabase security advisors report that `public.rls_auto_enable()` is executable by `anon` and `authenticated` while marked `SECURITY DEFINER`.

Catalog inspection confirmed:

- Owner: `postgres`
- `SECURITY DEFINER`: true
- Function setting: `search_path=pg_catalog`
- Explicit ACL includes `anon=X`, `authenticated=X`, and `service_role=X`
- It is registered as event trigger `ensure_rls` for `ddl_command_end`, limited to `CREATE TABLE`, `CREATE TABLE AS`, and `SELECT INTO`.
- Its body iterates over DDL commands and enables RLS on newly created tables in `public`; it does not read application rows or return table data.

This narrows the observed behavior: it is an RLS-enforcement event-trigger function, not a general-purpose data-access routine. However, public EXECUTE grants are unnecessary for an event-trigger callback and should be reviewed. The Supabase advisor's generic RPC exposure warning should be reconciled with the fact that the function returns the special `event_trigger` type.

**Recommended remediation for a separately approved change:**

- Confirm event-trigger operation remains intact after privilege changes.
- Revoke direct EXECUTE from `PUBLIC`, `anon`, and `authenticated` if confirmed not required; preserve only the minimum privileges needed for the trigger lifecycle and trusted administration.
- Do not change it to `SECURITY INVOKER` or drop the event trigger without testing migration behavior.
- Re-run security advisors and a controlled DDL test in staging before production.

No privilege changes were applied in this audit.

## 5. Priority and release gates

| Priority | Finding | Required evidence before closure |
|---|---|---|
| P0 | Active signing key / JWKS / real JWT claims unverified | Dashboard signing-key confirmation; JWKS public-key match; decoded non-secret token header and claims; verifier test |
| P1 | RLS policy design and actual CRM data access path unknown | Data-access architecture; table classification; per-operation policy tests |
| P1 | `rls_auto_enable()` has broad EXECUTE grants | Reviewed ACL change; event-trigger regression test; clean security advisor result or documented exception |
| P1 | Internal tables have broad table SELECT grants alongside RLS | Explicit grants review; confirm no privileged key or bypass path is exposed |
| P2 | IAM-03 through IAM-14 remain open | Maintainer-approved IAM decision record and requirement-mapped CI evidence |

## 6. Explicit non-actions

- No production RLS policies were created.
- No grants or revokes were applied.
- No functions or event triggers were altered.
- No Supabase Auth signing key, JWT template, hook, or token lifetime setting was changed.
- No production environment variables or secrets were changed.
- No schema migration was applied.

## 7. References

- Supabase RLS guide: https://supabase.com/docs/guides/database/postgres/row-level-security
- Supabase database linter — RLS enabled without policy: https://supabase.com/docs/guides/database/database-linter?lint=0008_rls_enabled_no_policy
- Supabase database linter — SECURITY DEFINER callable by anon: https://supabase.com/docs/guides/database/database-linter?lint=0028_anon_security_definer_function_executable
- Supabase database linter — SECURITY DEFINER callable by authenticated: https://supabase.com/docs/guides/database/database-linter?lint=0029_authenticated_security_definer_function_executable
