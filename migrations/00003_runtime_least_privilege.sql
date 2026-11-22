-- +goose Up

-- The migrator connection assumes naeos_intelligence_migrator before applying
-- migrations, so these objects are owned by the NOLOGIN group role.
GRANT USAGE ON SCHEMA naeos_intelligence TO naeos_intelligence_runtime;

GRANT SELECT, INSERT, UPDATE ON naeos_intelligence.intelligence_requests
    TO naeos_intelligence_runtime;
GRANT INSERT ON naeos_intelligence.request_transitions
    TO naeos_intelligence_runtime;
GRANT SELECT, INSERT ON naeos_intelligence.governance_audit_events
    TO naeos_intelligence_runtime;

-- Attempts and approval decisions are intentionally not granted to the API
-- runtime. They remain reserved for separately governed worker/approval roles.
REVOKE ALL ON naeos_intelligence.request_attempts,
              naeos_intelligence.approval_decisions,
              naeos_intelligence.goose_db_version
    FROM naeos_intelligence_runtime;

ALTER TABLE naeos_intelligence.intelligence_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE naeos_intelligence.intelligence_requests FORCE ROW LEVEL SECURITY;
ALTER TABLE naeos_intelligence.request_transitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE naeos_intelligence.request_transitions FORCE ROW LEVEL SECURITY;
ALTER TABLE naeos_intelligence.governance_audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE naeos_intelligence.governance_audit_events FORCE ROW LEVEL SECURITY;

CREATE POLICY intelligence_requests_owner_select
    ON naeos_intelligence.intelligence_requests
    FOR SELECT TO naeos_intelligence_runtime
    USING (client_id = NULLIF(current_setting('naeos.owner_id', true), ''));

CREATE POLICY intelligence_requests_owner_insert
    ON naeos_intelligence.intelligence_requests
    FOR INSERT TO naeos_intelligence_runtime
    WITH CHECK (client_id = NULLIF(current_setting('naeos.owner_id', true), ''));

CREATE POLICY intelligence_requests_owner_update
    ON naeos_intelligence.intelligence_requests
    FOR UPDATE TO naeos_intelligence_runtime
    USING (client_id = NULLIF(current_setting('naeos.owner_id', true), ''))
    WITH CHECK (client_id = NULLIF(current_setting('naeos.owner_id', true), ''));

CREATE POLICY request_transitions_owner_insert
    ON naeos_intelligence.request_transitions
    FOR INSERT TO naeos_intelligence_runtime
    WITH CHECK (
        EXISTS (
            SELECT 1
            FROM naeos_intelligence.intelligence_requests r
            WHERE r.id = request_id
              AND r.client_id = NULLIF(current_setting('naeos.owner_id', true), '')
        )
    );

CREATE POLICY governance_audit_events_owner_select
    ON naeos_intelligence.governance_audit_events
    FOR SELECT TO naeos_intelligence_runtime
    USING (client_id = NULLIF(current_setting('naeos.owner_id', true), ''));

CREATE POLICY governance_audit_events_owner_insert
    ON naeos_intelligence.governance_audit_events
    FOR INSERT TO naeos_intelligence_runtime
    WITH CHECK (client_id = NULLIF(current_setting('naeos.owner_id', true), ''));

-- +goose Down
DROP POLICY governance_audit_events_owner_insert ON naeos_intelligence.governance_audit_events;
DROP POLICY governance_audit_events_owner_select ON naeos_intelligence.governance_audit_events;
DROP POLICY request_transitions_owner_insert ON naeos_intelligence.request_transitions;
DROP POLICY intelligence_requests_owner_update ON naeos_intelligence.intelligence_requests;
DROP POLICY intelligence_requests_owner_insert ON naeos_intelligence.intelligence_requests;
DROP POLICY intelligence_requests_owner_select ON naeos_intelligence.intelligence_requests;

ALTER TABLE naeos_intelligence.governance_audit_events NO FORCE ROW LEVEL SECURITY;
ALTER TABLE naeos_intelligence.governance_audit_events DISABLE ROW LEVEL SECURITY;
ALTER TABLE naeos_intelligence.request_transitions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE naeos_intelligence.request_transitions DISABLE ROW LEVEL SECURITY;
ALTER TABLE naeos_intelligence.intelligence_requests NO FORCE ROW LEVEL SECURITY;
ALTER TABLE naeos_intelligence.intelligence_requests DISABLE ROW LEVEL SECURITY;

REVOKE ALL ON naeos_intelligence.intelligence_requests,
              naeos_intelligence.request_transitions,
              naeos_intelligence.governance_audit_events
    FROM naeos_intelligence_runtime;
