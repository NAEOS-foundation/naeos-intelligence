package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// RecordEvents appends a complete orchestration evidence batch atomically.
// Stable event IDs make a replay of the same request/event types idempotent;
// the transaction prevents partial batches when any insert fails.
func (s *Store) RecordEvents(ctx context.Context, clientID, requestID string, events []Event) error {
	if strings.TrimSpace(clientID) == "" || len(clientID) > 255 {
		return errors.New("client ID must contain 1 to 255 characters")
	}
	if !validUUID(requestID) {
		return errors.New("request ID must be a valid UUID")
	}
	if len(events) == 0 {
		return errors.New("at least one evidence event is required")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin evidence batch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for i, event := range events {
		if strings.TrimSpace(event.EventType) == "" || len(event.EventType) > 128 {
			return fmt.Errorf("event %d: event type must contain 1 to 128 characters", i)
		}
		if strings.TrimSpace(event.ActorType) == "" || len(event.ActorType) > 64 {
			return fmt.Errorf("event %d: actor type must contain 1 to 64 characters", i)
		}
		if strings.TrimSpace(event.ActorID) == "" || len(event.ActorID) > 255 {
			return fmt.Errorf("event %d: actor ID must contain 1 to 255 characters", i)
		}
		if strings.TrimSpace(event.Message) == "" || len(event.Message) > 4000 {
			return fmt.Errorf("event %d: message must contain 1 to 4000 characters", i)
		}
		if event.CorrelationID != "" && len(event.CorrelationID) > 255 {
			return fmt.Errorf("event %d: correlation ID must not exceed 255 characters", i)
		}

		id := deterministicEventID(clientID, requestID, event.EventType)
		_, err = tx.Exec(ctx, `
			INSERT INTO governance_audit_events
				(id, request_id, client_id, event_type, actor_type, actor_id, message, correlation_id)
			VALUES ($1,$2::uuid,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (id) DO NOTHING`,
			id, requestID, clientID, event.EventType, event.ActorType, event.ActorID,
			event.Message, nullIfBlank(event.CorrelationID))
		if err != nil {
			return fmt.Errorf("insert evidence event %d (%s): %w", i, event.EventType, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit evidence batch: %w", err)
	}
	return nil
}


func deterministicEventID(clientID, requestID, eventType string) string {
	sum := sha256.Sum256([]byte(clientID + "\x00" + requestID + "\x00" + eventType))
	id := sum[:16]
	// Mark the UUID as a name-based UUID (version 5) with the RFC 4122 variant.
	id[6] = (id[6] & 0x0f) | 0x50
	id[8] = (id[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(id)
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:]
}
