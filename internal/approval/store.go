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
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var requestIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}// Package approval provides durable PostgreSQL-backed persistence for
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
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

)

// validateRequestID rejects malformed UUIDs before PostgreSQL casts or queries.
func validateRequestID(requestID string) error {
	if strings.TrimSpace(requestID) == "" {
		return errors.New("request_id is required")
	}
	if !requestIDPattern.MatchString(requestID) {
		return errors.New("request_id must be a valid UUID")
	}
	return nil
}

// Store persists approval decisions to PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a durable approval store backed by the given connection pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Close releases the connection pool.
func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

// Ping verifies database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return errors.New("approval store database pool is nil")
	}
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("approval ping: %w", err)
	}
	return nil
}

// Evaluate determines whether a request needs approval. The latest decision
// is returned when one exists; otherwise a new decision is appended.
func (s *Store) Evaluate(ctx context.Context, requestID, executionMode, action string) (governance.ApprovalDecision, error) {
	trimmedMode := strings.TrimSpace(executionMode)
	trimmedAction := strings.TrimSpace(action)
	if err := validateRequestID(requestID); err != nil {
		return governance.ApprovalDecision{}, err
	}
	if trimmedAction == "" {
		return governance.ApprovalDecision{}, errors.New("action is required")
	}
	if trimmedMode == "" {
		trimmedMode = "analysis_only"
	}
	switch trimmedMode {
	case "analysis_only", "governed":
	default:
		return governance.ApprovalDecision{}, fmt.Errorf("unsupported execution_mode %q", executionMode)
	}
	if s == nil || s.pool == nil {
		return governance.ApprovalDecision{}, errors.New("approval store database pool is nil")
	}

	var existing governance.ApprovalDecision
	err := s.pool.QueryRow(ctx, `
		SELECT action, execution_mode, decision, approved_by, reason, created_at
		FROM approval_decisions WHERE request_id=$1::uuid
		ORDER BY created_at DESC, id DESC LIMIT 1
	`, requestID).Scan(&existing.Action, &existing.ExecutionMode, &existing.Decision,
		&existing.ApprovedBy, &existing.Reason, &existing.CreatedAt)
	if err == nil {
		if existing.Action != trimmedAction || existing.ExecutionMode != trimmedMode {
			return governance.ApprovalDecision{}, errors.New("request_id is already bound to a different action or execution_mode")
		}
		existing.RequestID = requestID
		existing.ApprovalRequired = existing.Decision == "pending"
		existing.CreatedAt = existing.CreatedAt.UTC()
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return governance.ApprovalDecision{}, fmt.Errorf("query existing decision: %w", err)
	}

	decision := governance.ApprovalDecision{
		RequestID: requestID, Action: trimmedAction, ExecutionMode: trimmedMode,
		CreatedAt: time.Now().UTC(),
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

// RecordDecision appends a human or system decision for a request. Existing
// records are never mutated; the latest row represents the current decision.
func (s *Store) RecordDecision(ctx context.Context, requestID, decision, approvedBy, reason string, at time.Time) error {
	if err := validateRequestID(requestID); err != nil {
		return err
	}
	decision = strings.TrimSpace(decision)
	if err := validateDecision(decision); err != nil {
		return err
	}
	if s == nil || s.pool == nil {
		return errors.New("approval store database pool is nil")
	}
	approvedBy = strings.TrimSpace(approvedBy)
	if approvedBy == "" {
		return errors.New("approved_by is required")
	}
	if len(approvedBy) > 255 {
		return errors.New("approved_by must be at most 255 characters")
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 4000 {
		return errors.New("reason must be at most 4000 characters")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin decision transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Serialize decisions per request even when no row exists yet.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", requestID); err != nil {
		return fmt.Errorf("lock request decision: %w", err)
	}

	action, executionMode := "system", "governed"
	var existingAction, existingMode, existingDecision string
	err = tx.QueryRow(ctx, `
		SELECT action, execution_mode, decision FROM approval_decisions
		WHERE request_id=$1::uuid ORDER BY created_at DESC, id DESC LIMIT 1
	`, requestID).Scan(&existingAction, &existingMode, &existingDecision)
	if err == nil {
		action, executionMode = existingAction, existingMode
		if existingDecision != "pending" {
			return fmt.Errorf("cannot record decision %q: request is already in terminal state %q", decision, existingDecision)
		}
		if decision == "pending" {
			return errors.New("cannot transition pending decision to pending")
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("query existing decision: %w", err)
	} else if decision != "pending" {
		return errors.New("initial decision must be pending")
	}

	id, err := newUUID()
	if err != nil {
		return fmt.Errorf("generate decision id: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO approval_decisions
			(id, request_id, client_id, action, execution_mode, decision, approved_by, reason, created_at)
		VALUES ($1,$2::uuid,'system',$3,$4,$5,$6,$7,$8)`,
		id, requestID, action, executionMode, decision, approvedBy, nullIfBlank(reason), at.UTC())
	if err != nil {
		return fmt.Errorf("insert decision: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit decision: %w", err)
	}
	return nil
}

// Get returns the latest recorded decision for a request ID if it exists.
func (s *Store) Get(ctx context.Context, requestID string) (governance.ApprovalDecision, bool) {
	if strings.TrimSpace(requestID) == "" || s == nil || s.pool == nil {
		return governance.ApprovalDecision{}, false
	}
	var d governance.ApprovalDecision
	err := s.pool.QueryRow(ctx, `
		SELECT action, execution_mode, decision, approved_by, reason, created_at
		FROM approval_decisions WHERE request_id=$1::uuid
		ORDER BY created_at DESC, id DESC LIMIT 1`, requestID).
		Scan(&d.Action, &d.ExecutionMode, &d.Decision, &d.ApprovedBy, &d.Reason, &d.CreatedAt)
	if err != nil {
		return governance.ApprovalDecision{}, false
	}
	d.RequestID = requestID
	d.ApprovalRequired = d.Decision == "pending"
	d.CreatedAt = d.CreatedAt.UTC()
	return d, true
}

// Decisions returns all recorded decisions sorted by timestamp.
func (s *Store) Decisions(ctx context.Context) ([]governance.ApprovalDecision, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("approval store database pool is nil")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT request_id::text, action, execution_mode, decision, approved_by, reason, created_at
		FROM approval_decisions ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query decisions: %w", err)
	}
	defer rows.Close()

	var items []governance.ApprovalDecision
	for rows.Next() {
		var d governance.ApprovalDecision
		if err := rows.Scan(&d.RequestID, &d.Action, &d.ExecutionMode, &d.Decision,
			&d.ApprovedBy, &d.Reason, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		d.ApprovalRequired = d.Decision == "pending"
		d.CreatedAt = d.CreatedAt.UTC()
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decisions: %w", err)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].RequestID < items[j].RequestID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func validateDecision(decision string) error {
	switch decision {
	case "pending", "approved", "rejected":
		return nil
	default:
		return fmt.Errorf("unsupported approval decision %q; must be pending, approved, or rejected", decision)
	}
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
