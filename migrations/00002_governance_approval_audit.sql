-- +goose Up
CREATE TABLE approval_decisions (
    id UUID PRIMARY KEY,
    request_id UUID NOT NULL UNIQUE REFERENCES intelligence_requests(id) ON DELETE RESTRICT,
    client_id TEXT NOT NULL CHECK (length(client_id) BETWEEN 1 AND 255),
    action TEXT NOT NULL CHECK (length(action) BETWEEN 1 AND 255),
    execution_mode TEXT NOT NULL CHECK (execution_mode = 'governed'),
    decision TEXT NOT NULL CHECK (decision IN ('approved', 'denied', 'rejected', 'pending')),
    approved_by TEXT NOT NULL CHECK (length(approved_by) BETWEEN 1 AND 255),
    reason TEXT CHECK (reason IS NULL OR length(reason) <= 4000),
    correlation_id TEXT CHECK (correlation_id IS NULL OR length(correlation_id) <= 255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX approval_decisions_client_created_idx ON approval_decisions (client_id, created_at DESC);

CREATE TABLE governance_audit_events (
    id UUID PRIMARY KEY,
    request_id UUID NOT NULL REFERENCES intelligence_requests(id) ON DELETE RESTRICT,
    client_id TEXT NOT NULL CHECK (length(client_id) BETWEEN 1 AND 255),
    event_type TEXT NOT NULL CHECK (length(event_type) BETWEEN 1 AND 128),
    actor_type TEXT NOT NULL CHECK (length(actor_type) BETWEEN 1 AND 64),
    actor_id TEXT NOT NULL CHECK (length(actor_id) BETWEEN 1 AND 255),
    message TEXT NOT NULL CHECK (length(message) <= 4000),
    correlation_id TEXT CHECK (correlation_id IS NULL OR length(correlation_id) <= 255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX governance_audit_events_request_created_idx ON governance_audit_events (request_id, created_at, id);
CREATE INDEX governance_audit_events_client_created_idx ON governance_audit_events (client_id, created_at DESC);

-- +goose StatementBegin
CREATE FUNCTION prevent_governance_audit_mutation() RETURNS trigger
LANGUAGE plpgsql AS 'BEGIN
    RAISE EXCEPTION ''governance audit events are append-only'';
END;';
-- +goose StatementEnd
CREATE TRIGGER governance_audit_events_no_update
    BEFORE UPDATE OR DELETE ON governance_audit_events
    FOR EACH ROW EXECUTE FUNCTION prevent_governance_audit_mutation();

-- +goose Down
DROP TRIGGER governance_audit_events_no_update ON governance_audit_events;
DROP FUNCTION prevent_governance_audit_mutation();
DROP TABLE governance_audit_events;
DROP TABLE approval_decisions;
