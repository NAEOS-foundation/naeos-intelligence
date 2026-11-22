# Unified Supabase Cutover Runbook — Draft

This runbook is a controlled operational checklist for the planned move of NAEOS Intelligence persistence into CRM production. It is **not an authorization to run the cutover**.

## Pre-flight
- [ ] Maintainer confirms `naeos.db` is staging and acknowledges that it currently contains the Intelligence schema/ledger-head row.
- [ ] Complete read-only inventory of Storage, roles/grants, schemas, routines, triggers, policies, migration metadata, repo references, deployment workflows, and secret *names*.
- [ ] Production CRM backup completed; restore procedure tested or evidenced.
- [ ] Source backup completed; recovery access confirmed.
- [ ] Migration SQL reviewed by a second pass; no destructive statements, no unqualified object names.
- [ ] Destination collision check confirms the `naeos_control_plane` schema does not already contain incompatible objects.
- [ ] IAM-03..14 security decisions are resolved for any auth activation; database consolidation alone does not close IAM.

## Staging rehearsal
- [ ] Rehearse schema creation and object deployment in a safe staging context.
- [ ] Apply constraints and indexes; validate all constraints.
- [ ] Apply explicit grants and RLS policies.
- [ ] Transfer rows transactionally.
- [ ] Compare table counts and schema definitions.
- [ ] Validate ledger chain/head integrity and immutability behavior.
- [ ] Test negative authorization paths.
- [ ] Run Intelligence tests and CRM-to-Intelligence API integration tests.
- [ ] Capture advisor findings and evidence artifacts.
- [ ] Record elapsed time, errors, and rollback demonstration.

## Production Gate P — must be approved explicitly
Approver: NAEOS maintainer. Required evidence: rehearsal report, backup evidence, DDL diff, grants/RLS matrix, reconciliation report, test results, and rollback steps.

## Cutover (only after Gate P)
1. Announce change window and stop/disable Intelligence write traffic.
2. Verify backup identifiers/timestamps and CRM healthy state.
3. Apply reviewed DDL to CRM production schema `naeos_control_plane`.
4. Transfer source data; reconcile expected row counts and ledger integrity.
5. Switch Intelligence runtime `DATABASE_URL` to dedicated destination connection secret.
6. Restart/redeploy Intelligence service through its normal deployment process.
7. Run health checks, protected API tests, ledger read/write tests as approved, and CRM integration smoke tests.
8. Monitor errors, database connections, security logs, and advisor findings.
9. Declare cutover complete only after all checks pass; keep source frozen as rollback.

## Rollback
- Stop Intelligence writes immediately.
- Restore prior Intelligence database connection.
- Restart service and run source-backed smoke tests.
- Preserve destination objects/data for investigation; do not drop.
- Reconcile post-cutover writes using a reviewed procedure.
- Escalate CRM production restore only if necessary and with explicit incident approval.

## Evidence record
- Change/PR:
- Approver:
- Window (UTC):
- CRM backup evidence:
- Source backup evidence:
- Migration commit:
- Schema diff:
- Row reconciliation:
- Ledger integrity:
- RLS/grant test:
- CI/test evidence:
- Post-cutover health:
- Rollback decision/result:
