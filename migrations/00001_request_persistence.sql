-- +goose Up
CREATE TABLE intelligence_requests (
    id UUID PRIMARY KEY,
    client_id TEXT NOT NULL CHECK (length(client_id) BETWEEN 1 AND 255),
    purpose TEXT NOT NULL CHECK (length(purpose) BETWEEN 1 AND 2000),
    execution_mode TEXT NOT NULL CHECK (execution_mode IN ('analysis_only', 'governed')),
    status TEXT NOT NULL CHECK (status IN ('accepted', 'queued', 'running', 'requires_approval', 'completed', 'failed', 'cancelled')),
    context_sources JSONB,
    desired_outputs JSONB,
    metadata JSONB,
    idempotency_key TEXT CHECK (idempotency_key IS NULL OR length(idempotency_key) BETWEEN 1 AND 255),
    payload_digest CHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    deadline_at TIMESTAMPTZ,
    CONSTRAINT requests_idempotency_client_key_unique UNIQUE (client_id, idempotency_key),
    CONSTRAINT requests_idempotency_digest_pair CHECK (
      (idempotency_key IS NULL AND payload_digest IS NULL) OR
      (idempotency_key IS NOT NULL AND payload_digest IS NOT NULL)
    )
);

CREATE INDEX intelligence_requests_client_created_idx
    ON intelligence_requests (client_id, created_at DESC);
CREATE INDEX intelligence_requests_status_created_idx
    ON intelligence_requests (status, created_at);

CREATE TABLE request_transitions (
    id UUID PRIMARY KEY,
    request_id UUID NOT NULL REFERENCES intelligence_requests(id) ON DELETE RESTRICT,
    from_status TEXT CHECK (from_status IS NULL OR from_status IN ('accepted', 'queued', 'running', 'requires_approval', 'completed', 'failed', 'cancelled')),
    to_status TEXT NOT NULL CHECK (to_status IN ('accepted', 'queued', 'running', 'requires_approval', 'completed', 'failed', 'cancelled')),
    event_type TEXT NOT NULL CHECK (length(event_type) BETWEEN 1 AND 128),
    actor_type TEXT NOT NULL CHECK (length(actor_type) BETWEEN 1 AND 64),
    actor_id TEXT,
    reason TEXT,
    correlation_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX request_transitions_request_created_idx
    ON request_transitions (request_id, created_at, id);

-- +goose StatementBegin
CREATE FUNCTION prevent_request_transition_mutation() RETURNS trigger
LANGUAGE plpgsql AS 'BEGIN
    RAISE EXCEPTION ''request transition history is append-only'';
END;';
-- +goose StatementEnd

CREATE TRIGGER request_transitions_no_update
    BEFORE UPDATE OR DELETE ON request_transitions
    FOR EACH ROW EXECUTE FUNCTION prevent_request_transition_mutation();

CREATE TABLE request_attempts (
    id UUID PRIMARY KEY,
    request_id UUID NOT NULL REFERENCES intelligence_requests(id) ON DELETE RESTRICT,
    attempt_number INTEGER NOT NULL CHECK (attempt_number > 0),
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'cancelled')),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    failure_category TEXT,
    failure_detail TEXT,
    worker_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT request_attempts_number_unique UNIQUE (request_id, attempt_number),
    CONSTRAINT request_attempts_time_order CHECK (finished_at IS NULL OR started_at IS NULL OR finished_at >= started_at)
);

CREATE INDEX request_attempts_request_idx ON request_attempts (request_id, attempt_number);

-- +goose Down
DROP TABLE request_attempts;
DROP TRIGGER request_transitions_no_update ON request_transitions;
DROP FUNCTION prevent_request_transition_mutation();
DROP TABLE request_transitions;
DROP TABLE intelligence_requests;
