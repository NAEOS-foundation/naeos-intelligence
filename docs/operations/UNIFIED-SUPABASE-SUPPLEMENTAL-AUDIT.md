# Unified Supabase Migration — Supplemental Read-only Audit Findings

**Status:** Read-only discovery; no database changes made.  
**Scope:** Supabase projects `kqmhpsftijskesjdovjm` (naeos.db; planned staging) and `oruykkehukoiipldzcvc` (CRM production; planned Intelligence destination).  
**Date:** 2026-11-16

## Storage inventory

- `storage.buckets` returned zero rows in both projects.
- No Storage bucket metadata or application object migration is presently indicated by this inventory. Confirm via application/API-level storage inventory if a future use-case adds buckets.

## CRM production — RLS and grants

- Security Advisor reports 23 public tables with RLS enabled and no policies (including `_prisma_migrations`).
- Read-only privilege aggregation found grants for `anon` and `authenticated` on 21 public tables across INSERT, SELECT, UPDATE, DELETE, TRUNCATE, REFERENCES, and TRIGGER privileges.
- This is a deny-by-default row posture while RLS has no policies for roles subject to RLS, but the broad table grants are unnecessary attack surface and should be reviewed against CRM's Prisma/service-only access model.
- Do not blanket-create policies. Establish role ownership, legitimate direct API use, and app-specific tenant/user authorization semantics first. Reconcile grants/policies in a dedicated security remediation plan and test API access with anon/authenticated/service roles.
- Public `rls_auto_enable()` is SECURITY DEFINER and pins `search_path` to `pg_catalog`; its trigger function body filters to `public` tables and logs failures. Review its event-trigger installation and exception behavior alongside the Prisma migration lifecycle.

## Intelligence control-plane schema — function security

- `naeos_control_plane.append_ledger_event(jsonb,text,text)` and `read_ledger_head()` are SECURITY DEFINER and pin `search_path` to `pg_catalog`.
- Direct EXECUTE privilege checks for anon, authenticated, and service_role returned false for both routines. This is a positive least-privilege signal, but must be validated against the intended dedicated runtime login/role and schema/table grants.
- Security Advisor reports `naeos_control_plane.ledger_head` with RLS enabled and no policy. Do not add a policy reflexively: the design appears to rely on privileged, controlled function access; verify dedicated runtime role has only required privileges and no direct public exposure.
- Keep this private schema isolated from CRM's Prisma-owned `public` schema.

## Cutover gates reinforced

1. Confirm dedicated Intelligence runtime role's exact schema/table/function privileges in CRM destination.
2. Ensure all runtime code and service composition actually use PostgreSQL persistence; the NAEOS repository ADR notes the demo composition can still be in-memory, and `DATABASE_URL` alone does not prove persistence.
3. Establish verified backup/restore evidence for CRM production and staging rehearsal; backup/restore evidence is not established by this read-only audit.
4. Verify RLS/policy and role access tests, migration drift, migration idempotency, ledger integrity and rollback plan.
5. No production DDL, data copy, secret changes, Auth configuration, deployment, or cutover until gates are reviewed and explicitly approved.

## Evidence references

- Supabase Security Advisor snapshots collected 2026-11-16.
- Read-only PostgreSQL catalog inspection of `pg_policies`, `information_schema.table_privileges`, and `pg_proc`/`pg_get_functiondef`.
- NAEOS repository public docs and migrations: `docs/architecture/ADR-PRODUCTION-CONTROL-PLANE-SUPABASE-PERSISTENCE.md`, `supabase/migrations/20261111000100_production_control_plane.sql`, and `supabase/migrations/20261111000200_control_plane_ledger_head_reader.sql`.
