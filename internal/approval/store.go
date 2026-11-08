// Package approval provides durable PostgreSQL-backed persistence for
// governed approval decisions. It implements the same decision contract as
// governance.ApprovalWorkflow but stores records durably in the
// approval_decisions table.
package approval

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists approval decisions to PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a durable approval store backed by the given connection pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Close releases the connection pool.
func (s *Store) Close() { s.pool.Close() }

// Ping verifies database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("approval ping: %w", err)
	}
	return nil
}

// Evaluate determines whether a request needs approval. If a decision already
// exists for the request ID, it is returned; otherwise a new pending decision
// is created for governed execution mode or an auto-approved decision for
// analysis_only mode.
func (s *Store) Evaluate(ctx context.Context, requestID, executionMode, action string) (governance.ApprovalDecision, error) {
	trimmedMode := strings.TrimSpace(executionMode)
	trimmedAction := strings.TrimSpace(action)
	if strings.TrimSpace(requestID) == "" {
		return governance.ApprovalDecision{}, errors.New("request_id is required")
	}
	if trimmedAction == "" {
		return governance.ApprovalDecision{}, errors.New("action is required")
	}
	if trimmedMode == "" {
		trimmedMode = "analysis_only"
	}

	// Validate mode before any DB access
	switch trimmedMode {
	case "analysis_only", "governed":
	default:
		return governance.ApprovalDecision{}, fmt.Errorf("unsupported execution_mode %q", executionMode)
	}

	// Check for existing decision
	var existingDecision, existingAction, existingMode, existingApprovedBy, existingReason string
	var existingCreatedAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT decision, action, execution_mode, approved_by, reason, created_at
		FROM approval_decisions WHERE request_id=$1::uuid
	`, requestID).Scan(&existingDecision, &existingAction, &existingMode, &existingApprovedBy, &existingReason, &existingCreatedAt)
	if err == nil {
		return governance.ApprovalDecision{
			RequestID:        requestID,
			Action:           existingAction,
			ExecutionMode:    existingMode,
			Decision:         existingDecision,
			ApprovedBy:       existingApprovedBy,
			Reason:           existingReason,
			ApprovalRequired: existingDecision == "pending",
			CreatedAt:        existingCreatedAt.UTC(),
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return governance.ApprovalDecision{}, fmt.Errorf("query existing decision: %w", err)
	}

	// No existing decision; create one based on execution mode
	decision := governance.ApprovalDecision{
		RequestID:     requestID,
		Action:        trimmedAction,
		ExecutionMode: trimmedMode,
		CreatedAt:     time.Now().UTC(),
	}
	switch trimmedMode {
	case "analysis_only":
		decision.Decision = "approved"
		decision.ApprovalRequired = false
		decision.Reason = "read-only analysis is permitted without additional approval"
	case "governed":
		decision.Decision = "pending"
		decision.ApprovalRequired = true
		decision.Reason = "governed execution requires a separately approved approval flow"
	}

	id, err := newUUID()
	if err != nil {
		return governance.ApprovalDecision{}, fmt.Errorf("generate decision id: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO approval_decisions
			(id, request_id, client_id, action, execution_mode, decision, approved_by, reason)
		VALUES ($1,$2::uuid,'system',$3,$4,$5,'system',$6)`,
		id, requestID, trimmedAction, trimmedMode, decision.Decision, nullIfBlank(decision.Reason))
	if err != nil {
		return governance.ApprovalDecision{}, fmt.Errorf("insert decision: %w", err)
	}

	return decision, nil
}

// RecordDecision records or updates a human or system decision for a request.
func (s *Store) RecordDecision(ctx context.Context, requestID, decision, approvedBy, reason string, at time.Time) error {
	if strings.TrimSpace(requestID) == "" {
		return errors.New("request_id is required")
	}
	if strings.TrimSpace(decision) == "" {
		return errors.New("decision is required")
	}
	decision = strings.TrimSpace(decision)
	approvedBy = strings.TrimSpace(approvedBy)
	reason = strings.TrimSpace(reason)
	if at.IsZero() {
		at = time.Now().UTC()
	}

	// Check if a decision already exists
	var existingDecision string
	err := s.pool.QueryRow(ctx, `SELECT decision FROM approval_decisions WHERE request_id=$1::uuid`, requestID).Scan(&existingDecision)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("query existing decision: %w", err)
		}
		// No existing decision; insert new
		id, idErr := newUUID()
		if idErr != nil {
			return fmt.Errorf("generate decision id: %w", idErr)
		}
		_, err = s.pool.Exec(ctx, `
			INSERT INTO approval_decisions
				(id, request_id, client_id, action, execution_mode, decision, approved_by, reason, created_at)
			VALUES ($1,$2::uuid,'system','system','governed',$3,$4,$5,$6)`,
			id, requestID, decision, approvedBy, reason, at.UTC())
		if err != nil {
			return fmt.Errorf("insert decision: %w", err)
		}
		return nil
	}

	// Existing decision; update it
	_, err = s.pool.Exec(ctx, `
		UPDATE approval_decisions
		SET decision=$1, approved_by=$2, reason=$3, created_at=$4
		WHERE request_id=$5::uuid`,
		decision, approvedBy, nullIfBlank(reason), at.UTC(), requestID)
	if err != nil {
		return fmt.Errorf("update decision: %w", err)
	}
	return nil
}

// Get returns a recorded decision for a request ID if it exists.
func (s *Store) Get(ctx context.Context, requestID string) (governance.ApprovalDecision, bool) {
	if strings.TrimSpace(requestID) == "" {
		return governance.ApprovalDecision{}, false
	}
	var d governance.ApprovalDecision
	var action, mode, decision, approvedBy, reason string
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT action, execution_mode, decision, approved_by, reason, created_at
		FROM approval_decisions WHERE request_id=$1::uuid`, requestID).
		Scan(&action, &mode, &decision, &approvedBy, &reason, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return governance.ApprovalDecision{}, false
		}
		return governance.ApprovalDecision{}, false
	}
	d.RequestID = requestID
	d.Action = action
	d.ExecutionMode = mode
	d.Decision = decision
	d.ApprovedBy = approvedBy
	d.Reason = reason
	d.ApprovalRequired = decision == "pending"
	d.CreatedAt = createdAt.UTC()
	return d, true
}

// Decisions returns all recorded decisions sorted by timestamp.
func (s *Store) Decisions(ctx context.Context) ([]governance.ApprovalDecision, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT request_id::text, action, execution_mode, decision, approved_by, reason, created_at
		FROM approval_decisions
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("query decisions: %w", err)
	}
	defer rows.Close()

	var items []governance.ApprovalDecision
	for rows.Next() {
		var d governance.ApprovalDecision
		var action, mode, decision, approvedBy, reason string
		var createdAt time.Time
		if err := rows.Scan(&d.RequestID, &action, &mode, &decision, &approvedBy, &reason, &createdAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		d.Action = action
		d.ExecutionMode = mode
		d.Decision = decision
		d.ApprovedBy = approvedBy
		d.Reason = reason
		d.ApprovalRequired = decision == "pending"
		d.CreatedAt = createdAt.UTC()
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decisions: %w", err)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:], nil
}

func nullIfBlank(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
