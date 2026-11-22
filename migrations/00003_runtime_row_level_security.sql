-- +goose Up

-- Runtime ownership is derived from the verified server-side principal and set
-- transaction-locally as naeos.owner_id. Missing context fails closed.
ALTER TABLE intelligence_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE intelligence_requests FORCE ROW LEVEL SECURITY;
CREATE POLICY intelligence_requests_owner_select
    ON intelligence_requests FOR SELECT TO naeos_intelligence_runtime
    USING (client_id = current_setting('naeos.owner_id', true));
CREATE POLICY intelligence_requests_owner_insert
    ON intelligence_requests FOR INSERT TO naeos_intelligence_runtime
    WITH CHECK (client_id = current_setting('naeos.owner_id', true));
CREATE POLICY intelligence_requests_owner_update
    ON intelligence_requests FOR UPDATE TO naeos_intelligence_runtime
    USING (client_id = current_setting('naeos.owner_id', true))
    WITH CHECK (client_id = current_setting('naeos.owner_id', true));

ALTER TABLE request_transitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE request_transitions FORCE ROW LEVEL SECURITY;
CREATE POLICY request_transitions_owner_insert
    ON request_transitions FOR INSERT TO naeos_intelligence_runtime
    WITH CHECK (
        EXISTS (
            SELECT 1
            FROM intelligence_requests r
            WHERE r.id = request_id
              AND r.client_id = current_setting('naeos.owner_id', true)
        )
    );

-- Worker attempts and approval decisions are deliberately unavailable to the
-- public API runtime. They require separately authorized worker/admin roles.
ALTER TABLE request_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE request_attempts FORCE ROW LEVEL SECURITY;
ALTER TABLE approval_decisions ENABLE ROW LEVEL SECURITY;
ALTER TABLE approval_decisions FORCE ROW LEVEL SECURITY;

ALTER TABLE governance_audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE governance_audit_events FORCE ROW LEVEL SECURITY;
CREATE POLICY governance_audit_events_owner_select
    ON governance_audit_events FOR SELECT TO naeos_intelligence_runtime
    USING (
        client_id = current_setting('naeos.owner_id', true)
        AND EXISTS (
            SELECT 1
            FROM intelligence_requests r
            WHERE r.id = request_id
              AND r.client_id = current_setting('naeos.owner_id', true)
        )
    );
CREATE POLICY governance_audit_events_owner_insert
    ON governance_audit_events FOR INSERT TO naeos_intelligence_runtime
    WITH CHECK (
        client_id = current_setting('naeos.owner_id', true)
        AND EXISTS (
            SELECT 1
            FROM intelligence_requests r
            WHERE r.id = request_id
              AND r.client_id = current_setting('naeos.owner_id', true)
        )
    );

REVOKE ALL ON ALL TABLES IN SCHEMA naeos_intelligence FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA naeos_intelligence FROM naeos_intelligence_runtime;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA naeos_intelligence FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA naeos_intelligence FROM naeos_intelligence_runtime;
GRANT USAGE ON SCHEMA naeos_intelligence TO naeos_intelligence_runtime;
REVOKE CREATE ON SCHEMA naeos_intelligence FROM naeos_intelligence_runtime;
GRANT SELECT, INSERT, UPDATE ON intelligence_requests TO naeos_intelligence_runtime;
GRANT INSERT ON request_transitions TO naeos_intelligence_runtime;
GRANT SELECT, INSERT ON governance_audit_events TO naeos_intelligence_runtime;
REVOKE ALL ON request_attempts, approval_decisions, goose_db_version FROM naeos_intelligence_runtime;

-- +goose Down
REVOKE ALL ON request_attempts, approval_decisions, goose_db_version FROM naeos_intelligence_runtime;
REVOKE SELECT, INSERT ON governance_audit_events FROM naeos_intelligence_runtime;
REVOKE INSERT ON request_transitions FROM naeos_intelligence_runtime;
REVOKE SELECT, INSERT, UPDATE ON intelligence_requests FROM naeos_intelligence_runtime;
REVOKE USAGE ON SCHEMA naeos_intelligence FROM naeos_intelligence_runtime;

DROP POLICY governance_audit_events_owner_insert ON governance_audit_events;
DROP POLICY governance_audit_events_owner_select ON governance_audit_events;
ALTER TABLE governance_audit_events NO FORCE ROW LEVEL SECURITY;
ALTER TABLE governance_audit_events DISABLE ROW LEVEL SECURITY;

ALTER TABLE approval_decisions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE approval_decisions DISABLE ROW LEVEL SECURITY;
ALTER TABLE request_attempts NO FORCE ROW LEVEL SECURITY;
ALTER TABLE request_attempts DISABLE ROW LEVEL SECURITY;

DROP POLICY request_transitions_owner_insert ON request_transitions;
ALTER TABLE request_transitions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE request_transitions DISABLE ROW LEVEL SECURITY;

DROP POLICY intelligence_requests_owner_update ON intelligence_requests;
DROP POLICY intelligence_requests_owner_insert ON intelligence_requests;
DROP POLICY intelligence_requests_owner_select ON intelligence_requests;
ALTER TABLE intelligence_requests NO FORCE ROW LEVEL SECURITY;
ALTER TABLE intelligence_requests DISABLE ROW LEVEL SECURITY;
