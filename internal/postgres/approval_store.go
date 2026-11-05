package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/httpapi"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/jackc/pgx/v5"
)

// RecordApprovalDecision durably records a governed request decision, updates its
// lifecycle state, and appends an audit event in a single PostgreSQL transaction.
// Repeating an identical decision is idempotent; a conflicting repeat is rejected.
func (s *Store) RecordApprovalDecision(ctx context.Context, clientID, requestID, decision, approvedBy, reason, correlationID string) (governance.ApprovalDecision, error) {
	clientID = strings.TrimSpace(clientID)
	decision = strings.ToLower(strings.TrimSpace(decision))
	approvedBy = strings.TrimSpace(approvedBy)
	reason = strings.TrimSpace(reason)
	if clientID == "" || len(clientID) > 255 {
		return governance.ApprovalDecision{}, errors.New("client ID must contain 1 to 255 characters")
	}
	if !validUUID(requestID) {
		return governance.ApprovalDecision{}, httpapi.ErrNotFound
	}
	if approvedBy == "" || len(approvedBy) > 255 {
		return governance.ApprovalDecision{}, errors.New("approved_by must contain 1 to 255 characters")
	}
	if len(reason) > 4000 || len(correlationID) > 255 {
		return governance.ApprovalDecision{}, errors.New("reason or correlation ID exceeds allowed length")
	}
	switch decision {
	case "approved", "denied", "rejected", "pending":
	default:
		return governance.ApprovalDecision{}, errors.New("decision must be approved, denied, rejected, or pending")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return governance.ApprovalDecision{}, err
	}
	defer tx.Rollback(ctx)

	var status, executionMode, action string
	var createdAt time.Time
	err = tx.QueryRow(ctx, `SELECT status, execution_mode, purpose, created_at
		FROM intelligence_requests WHERE client_id=$1 AND id=$2::uuid FOR UPDATE`,
		clientID, requestID).Scan(&status, &executionMode, &action, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return governance.ApprovalDecision{}, httpapi.ErrNotFound
	}
	if err != nil {
		return governance.ApprovalDecision{}, err
	}
	if executionMode != "governed" {
		return governance.ApprovalDecision{}, fmt.Errorf("%w: approval is only valid for governed requests", httpapi.ErrConflict)
	}

	var existing governance.ApprovalDecision
	var existingReason string
	err = tx.QueryRow(ctx, `SELECT request_id::text, action, execution_mode, decision, approved_by, COALESCE(reason,''), created_at
		FROM approval_decisions WHERE client_id=$1 AND request_id=$2::uuid`,
		clientID, requestID).Scan(&existing.RequestID, &existing.Action, &existing.ExecutionMode,
		&existing.Decision, &existing.ApprovedBy, &existingReason, &existing.CreatedAt)
	if err == nil {
		if existing.Decision != decision || existing.ApprovedBy != approvedBy || existingReason != reason {
			return governance.ApprovalDecision{}, httpapi.ErrConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return governance.ApprovalDecision{}, err
		}
		existing.Reason = existingReason
		existing.ApprovalRequired = existing.Decision == "pending"
		existing.CreatedAt = existing.CreatedAt.UTC()
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return governance.ApprovalDecision{}, err
	}

	if status != string(httpapi.StatusAccepted) && status != string(httpapi.StatusRequiresApproval) {
		return governance.ApprovalDecision{}, httpapi.ErrConflict
	}
	decisionID, err := newUUID()
	if err != nil {
		return governance.ApprovalDecision{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO approval_decisions
		(id, request_id, client_id, action, execution_mode, decision, approved_by, reason)
		VALUES ($1,$2::uuid,$3,$4,'governed',$5,$6,$7)`,
		decisionID, requestID, clientID, action, decision, approvedBy, nullIfBlank(reason))
	if err != nil {
		return governance.ApprovalDecision{}, err
	}

	nextStatus := status
	eventType := "approval.decision_recorded"
	if decision == "approved" {
		nextStatus = string(httpapi.StatusQueued)
	} else if decision == "denied" || decision == "rejected" {
		nextStatus = string(httpapi.StatusCancelled)
	}
	if nextStatus != status {
		_, err = tx.Exec(ctx, `UPDATE intelligence_requests SET status=$3, updated_at=now()
			WHERE client_id=$1 AND id=$2::uuid`, clientID, requestID, nextStatus)
		if err != nil {
			return governance.ApprovalDecision{}, err
		}
		transitionID, idErr := newUUID()
		if idErr != nil {
			return governance.ApprovalDecision{}, idErr
		}
		_, err = tx.Exec(ctx, `INSERT INTO request_transitions
			(id, request_id, from_status, to_status, event_type, actor_type, actor_id, reason, correlation_id)
			VALUES ($1,$2::uuid,$3,$4,$5,'human',$6,$7,$8)`,
			transitionID, requestID, status, nextStatus, eventType, approvedBy,
			nullIfBlank(reason), nullIfBlank(correlationID))
		if err != nil {
			return governance.ApprovalDecision{}, err
		}
	}

	auditID, err := newUUID()
	if err != nil {
		return governance.ApprovalDecision{}, err
	}
	message := "approval decision: " + decision
	if reason != "" {
		message += "; reason: " + reason
	}
	_, err = tx.Exec(ctx, `INSERT INTO governance_audit_events
		(id, request_id, client_id, event_type, actor_type, actor_id, message, correlation_id)
		VALUES ($1,$2::uuid,$3,'approval_decision_recorded','human',$4,$5,$6)`,
		auditID, requestID, clientID, approvedBy, message, nullIfBlank(correlationID))
	if err != nil {
		return governance.ApprovalDecision{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return governance.ApprovalDecision{}, err
	}
	return governance.ApprovalDecision{
		RequestID: requestID, Action: action, ExecutionMode: executionMode,
		Decision: decision, ApprovedBy: approvedBy, Reason: reason,
		ApprovalRequired: decision == "pending", CreatedAt: createdAt.UTC(),
	}, nil
}
