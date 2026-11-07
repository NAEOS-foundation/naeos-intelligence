// Package evidence implements durable evidence and audit persistence
// backed by PostgreSQL governance_audit_events table.
package evidence

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Event is a durable security-relevant audit record.
type Event struct {
	ID            string    `json:"id"`
	RequestID     string    `json:"request_id"`
	ClientID      string    `json:"client_id"`
	EventType     string    `json:"event_type"`
	ActorType     string    `json:"actor_type"`
	ActorID       string    `json:"actor_id"`
	Message       string    `json:"message"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Store persists evidence events to PostgreSQL.
type Store struct{ pool *pgxpool.Pool }

// New creates a durable evidence store backed by the given connection pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

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

// RecordEvent appends an audit event to governance_audit_events.
// Events are append-only; UPDATE/DELETE is blocked by a database trigger.
func (s *Store) RecordEvent(ctx context.Context, clientID, requestID, eventType, actorType, actorID, message, correlationID string) error {
	if strings.TrimSpace(clientID) == "" || len(clientID) > 255 {
		return errors.New("client ID must contain 1 to 255 characters")
	}
	if !validUUID(requestID) {
		return errors.New("request ID must be a valid UUID")
	}
	if strings.TrimSpace(eventType) == "" || len(eventType) > 128 {
		return errors.New("event type must contain 1 to 128 characters")
	}
	if strings.TrimSpace(actorType) == "" || len(actorType) > 64 {
		return errors.New("actor type must contain 1 to 64 characters")
	}
	if strings.TrimSpace(actorID) == "" || len(actorID) > 255 {
		return errors.New("actor ID must contain 1 to 255 characters")
	}
	if strings.TrimSpace(message) == "" || len(message) > 4000 {
		return errors.New("message must contain 1 to 4000 characters")
	}
	if correlationID != "" && len(correlationID) > 255 {
		return errors.New("correlation ID must not exceed 255 characters")
	}

	id, err := newUUID()
	if err != nil {
		return fmt.Errorf("generate event ID: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO governance_audit_events
			(id, request_id, client_id, event_type, actor_type, actor_id, message, correlation_id)
		VALUES ($1,$2::uuid,$3,$4,$5,$6,$7,$8)`,
		id, requestID, clientID, eventType, actorType, actorID, message, nullIfBlank(correlationID))
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

// GetEvents returns all events for a request, ordered by created_at ascending.
func (s *Store) GetEvents(ctx context.Context, clientID, requestID string) ([]Event, error) {
	if strings.TrimSpace(clientID) == "" || len(clientID) > 255 {
		return nil, errors.New("client ID must contain 1 to 255 characters")
	}
	if !validUUID(requestID) {
		return nil, errors.New("request ID must be a valid UUID")
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id::text, request_id::text, client_id, event_type, actor_type, actor_id, message,
		       correlation_id, created_at
		FROM governance_audit_events
		WHERE client_id=$1 AND request_id=$2::uuid
		ORDER BY created_at ASC, id ASC`,
		clientID, requestID)
	if err != nil {
		return nil, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.RequestID, &e.ClientID, &e.EventType, &e.ActorType,
			&e.ActorID, &e.Message, &e.CorrelationID, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		e.CreatedAt = e.CreatedAt.UTC()
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}
	return events, nil
}

// Close releases the connection pool.
func (s *Store) Close() { s.pool.Close() }

// Ping verifies database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("evidence ping: %w", err)
	}
	return nil
}

func validUUID(v string) bool {
	if len(v) != 36 || v[8] != '-' || v[13] != '-' || v[18] != '-' || v[23] != '-' {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(v, "-", ""))
	return err == nil
}
