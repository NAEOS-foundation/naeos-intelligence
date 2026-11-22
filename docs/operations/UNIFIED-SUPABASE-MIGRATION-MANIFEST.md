# Unified Supabase Migration Manifest — Draft

**Status:** Planning artifact only; no database DDL or data transfer authorized by this file.
**Target:** Consolidate NAEOS Intelligence persistence into `naeos-crm-production` (`oruykkehukoiipldzcvc`, Singapore).
**Temporary staging:** `naeos.db` (`kqmhpsftijskesjdovjm`, Tokyo).
**Decision owner:** NAEOS maintainer.

## 1. Current inventory baseline

### Intelligence source (`naeos.db`)
Schema: `naeos_control_plane`; 6 tables:
- `active_policies` — 0 rows
- `policy_versions` — 0 rows
- `grants` — 0 rows
- `approvals` — 0 rows
- `ledger_events` — 0 rows
- `ledger_head` — 1 row

Other observed objects: primary/foreign keys; indexes for agent/request/decision/time and relationships; RLS enabled on all six tables; six RLS policies; four custom routines; two immutability triggers; three Supabase migration records. Advisor findings: `ledger_head` has RLS enabled without a policy; two foreign keys are reported without covering indexes. Resolve and document these explicitly, not by broad policy grants.

### CRM destination (`naeos-crm-production`)
Schema `public`: 22 application tables plus `_prisma_migrations`. Four application rows in total at inventory time (User=1, AuditEvent=3); migration history has 12 records. Prisma owns the CRM application schema. CRM Security Advisor flags all 23 public tables with RLS enabled and no policies; current grants must be reviewed with the API's backend access pattern. Do not weaken or repurpose CRM policies to accommodate Intelligence.

### Naming boundary
Keep Intelligence objects in `naeos_control_plane`. Do not merge into `public`; CRM already has domain objects named `PolicyRule`, `PolicyDecision`, `GoGateRequest`, and `EvidenceReceipt`, which are not interchangeable with the Intelligence tables.

## 2. Migration object manifest

| Object class | Source object set | Destination handling | Validation / gate |
|---|---|---|---|
| Schema | `naeos_control_plane` | Create same schema in CRM; retain schema ownership boundary | Schema exists; ownership and search_path reviewed |
| Tables | 6 named above | Recreate from source DDL; preserve column types/defaults/nullability | Column-by-column diff; row counts |
| Constraints | PK, FK, unique/check constraints | Recreate exactly; validate references | Constraint inventory and validation query |
| Indexes | Source indexes | Recreate; include missing FK-covering indexes only after query-plan review | Index definition diff; advisor review |
| RLS | Enablement + 6 policies | Recreate policies verbatim initially, then security review | Test as anon/authenticated/service_role and app roles; fail closed |
| Grants | Schema/table/function privileges | Explicit least-privilege grants; do not copy blanket grants | Effective privilege matrix reviewed |
| Routines | 4 custom routines | Recreate exact definitions, dependencies, security-definer settings, search_path | Definition diff; unit/integration tests |
| Triggers | 2 immutability triggers | Recreate exact trigger/function binding | Trigger inventory; immutable ledger tests |
| Data | 1 ledger_head row; other tables empty | Transfer transactionally after schema verification | Source/target count and ledger-head integrity |
| Migration metadata | 3 Supabase migration records | Decide metadata strategy; do not blindly insert into destination migration history | Migration tool ownership and collision review |
| Extensions | pg_stat_statements 1.11, pgcrypto 1.3, plpgsql 1.0, supabase_vault 0.3.1, uuid-ossp 1.1 on both | Confirm availability/version; no unnecessary extension changes | Extension inventory matches |
| Storage / Auth / Realtime | Not yet fully inventoried for this cutover | Out of scope until dependency review | Inventory and explicit decision required |

## 3. Dependency and configuration manifest

| Consumer | Current setting | Planned value | Gate |
|---|---|---|---|
| Intelligence service | `DATABASE_URL` currently points at Intelligence DB | CRM production connection, using a dedicated least-privilege database role | staging rehearsal + role review + approved cutover |
| CRM API | CRM's `DATABASE_URL` | Remains CRM production; Prisma continues to own CRM public schema | no change |
| CRM → Intelligence | `INTELLIGENCE_API_URL`, `INTELLIGENCE_SERVICE_TOKEN` | API boundary remains; token permissions audited | service token rotation/verification |
| Intelligence Auth | `NAEOS_AUTH_ISSUER`, `NAEOS_AUTH_AUDIENCE`, `NAEOS_AUTH_JWKS_URL` | Candidate issuer is CRM Supabase Auth; values remain unset until IAM gates accepted | IAM-03..14 + real-token/JWKS validation |
| Local/dev | CRM local Postgres; Intelligence dev Postgres | Keep isolated; never point local/dev to production | environment audit |
| Deploy/workflows/secrets | Repo, CI, hosting secret names not fully inventoried | Enumerate exact consumers and secret names; never export secret values into docs | dependency closure complete |

## 4. Required sequence (no production writes until Gate P)

1. **Freeze source inventory:** obtain complete source DDL for schema, policies, grants, routines, triggers, indexes, sequences, migration history; inspect Storage buckets and app/deployment references.
2. **Backup/recovery:** take an approved CRM production backup and source backup; establish restore path and record backup timestamps. No backup/restore evidence exists yet.
3. **Build migration SQL:** generate reviewed, deterministic schema migration. Include explicit schema qualification and avoid implicit `search_path`.
4. **Staging rehearsal:** apply only to `naeos.db` as staging after confirming it is safe to use as rehearsal target; do not erase existing source objects. Prefer isolated disposable test schema/database if collisions exist.
5. **Transfer data:** copy data with transaction, preserving ledger identity/hash sequence; reconcile counts and cryptographic/integrity checks.
6. **Security rehearsal:** test RLS, grants, function execution, service role behavior, auth identity mapping, denied cases, and audit attribution.
7. **Application rehearsal:** run Intelligence tests against staging destination; verify CRM integration contract and no Prisma drift.
8. **Gate P (production approval):** maintainer explicitly approves production DDL/data transfer after reviewing rehearsal evidence and rollback plan.
9. **Cutover:** create destination schema/objects, transfer data, validate, update only the Intelligence runtime's database connection. Keep CRM Prisma config unchanged.
10. **Post-cutover:** smoke test, inspect logs/advisors, retain source in a frozen rollback state; no project deletion or source cleanup without separate approval.

## 5. Acceptance criteria

- [ ] Exact DDL/object inventory captured and versioned.
- [ ] Source and target backup/restore procedure verified, with evidence.
- [ ] Table/column/constraint/index/routine/trigger/policy/grant diff reviewed.
- [ ] Row counts reconcile (expected source baseline: 1 total row).
- [ ] Ledger head and chain/hash integrity check passes.
- [ ] RLS and role tests include allowed and denied scenarios; no broad public access.
- [ ] Security and Performance Advisors reviewed; findings dispositioned.
- [ ] Intelligence automated tests, integration tests, and smoke tests pass.
- [ ] CRM auth and CRM public schema remain unchanged.
- [ ] IAM-03 through IAM-14 are accepted before production auth configuration.
- [ ] Rollback criteria, operator, and procedure documented.
- [ ] Production cutover separately approved.

## 6. Rollback

Rollback triggers: failed data reconciliation, ledger integrity failure, unexpected privilege/RLS exposure, failed app smoke tests, or unacceptable production error rate.

Rollback plan: stop Intelligence writes; restore its prior database connection to the original source; preserve CRM destination objects/data for forensic review (do not drop); reconcile any writes that occurred after cutover under an explicitly reviewed recovery procedure. A point-in-time restore of CRM production is a last resort and requires separate incident-level approval because it can affect CRM records.

## 7. Explicit non-actions

This plan does not execute DDL, copy records, change secrets, alter CRM Auth, enable IAM production values, drop objects, pause projects, or merge/delete any Supabase project.
