# Canonical Human Principal and Storage Ownership

**Status:** Implementation proposal on `feat/iam-canonical-issuer-subject-v1`  
**Scope:** Non-production API identity and ownership boundary. No database schema migration, backfill, Supabase configuration, or deployment is included.

## Canonical identity

- A human principal is identified by the exact verified JWT issuer (`iss`) and subject (`sub`) tuple.
- The issuer namespace is part of the identity. Equal `sub` values from different issuers must never collide.
- OAuth `client_id` remains a separate attribute. It is not a fallback for a missing human subject and is not the storage owner.
- This implementation is human-session-only. Machine/service principals remain unsupported until a separate typed principal and authorization contract is accepted. Tokens without a non-empty `sub` are rejected at the protected API boundary.

## Persistence compatibility

The current persistence schema calls its owner column `client_id`. This change deliberately avoids a DDL migration and stores an opaque, deterministic owner key in that existing column:

`user:<base64url(iss)>:<base64url(sub)>`

The key is server-derived after JWT verification and is never accepted from request bodies or headers. The composite is bounded to the existing 255-character column constraint. All handler ownership reads, writes, cancellation, analysis, and idempotency keys use this same derived owner key.

The column name is now a compatibility artifact, not a statement that the OAuth client owns the record. A future schema migration may rename it to `owner_key`, but that requires a separate migration and compatibility plan.

## Existing records

Existing rows keyed by legacy raw `client_id` are not rewritten or automatically assigned to a human subject. This change does not claim that a newly authenticated human can access legacy client-owned records. Any legacy ownership mapping or backfill requires an explicit, evidence-backed migration decision covering requests, idempotency uniqueness, evidence, approvals, transitions, and audit attribution.

## Verification requirements

- Same `sub` under different `iss` values produces different owner keys.
- Same `client_id` with different `iss + sub` tuples produces independent request and idempotency records.
- Missing issuer, missing/blank subject, or oversized composite key fails closed.
- Cross-owner GET/CANCEL/ANALYZE returns the same not-found response.
- The owner key is derived only from verified server-side principal context.
- Existing client-only/service tokens are not silently treated as human principals.
- No public approval endpoint is introduced by this phase.

## Explicit non-actions

No schema DDL, data movement, backfill, Supabase Auth hook, JWT algorithm migration, revocation lookup, staging deployment, or production change is performed by this branch.
