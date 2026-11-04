package governance

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ApprovalDecision captures a policy decision for a future governed workflow.
type ApprovalDecision struct {
	RequestID        string    `json:"request_id"`
	Action           string    `json:"action"`
	ExecutionMode    string    `json:"execution_mode"`
	Decision         string    `json:"decision"`
	ApprovedBy       string    `json:"approved_by,omitempty"`
	Reason           string    `json:"reason,omitempty"`
	ApprovalRequired bool      `json:"approval_required"`
	CreatedAt        time.Time `json:"created_at"`
}

// ApprovalWorkflow stores policy-aligned approval state for future use.
type ApprovalWorkflow struct {
	mu        sync.RWMutex
	decisions map[string]ApprovalDecision
}

// NewApprovalWorkflow creates an in-memory approval registry for the read-only MVP scaffold.
func NewApprovalWorkflow() *ApprovalWorkflow {
	return &ApprovalWorkflow{decisions: make(map[string]ApprovalDecision)}
}

// Evaluate determines whether a request needs approval in a future governed execution phase.
func (w *ApprovalWorkflow) Evaluate(requestID, executionMode, action string) (ApprovalDecision, error) {
	trimmedMode := strings.TrimSpace(executionMode)
	trimmedAction := strings.TrimSpace(action)
	if strings.TrimSpace(requestID) == "" {
		return ApprovalDecision{}, errors.New("request_id is required")
	}
	if trimmedAction == "" {
		return ApprovalDecision{}, errors.New("action is required")
	}
	if trimmedMode == "" {
		trimmedMode = "analysis_only"
	}
	decision := ApprovalDecision{
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
	default:
		return ApprovalDecision{}, fmt.Errorf("unsupported execution_mode %q", executionMode)
	}
	if w != nil {
		w.mu.Lock()
		w.decisions[requestID] = decision
		w.mu.Unlock()
	}
	return decision, nil
}

// RecordDecision records a human or system outcome for an existing request.
func (w *ApprovalWorkflow) RecordDecision(requestID, decision, approvedBy, reason string, at time.Time) error {
	if strings.TrimSpace(requestID) == "" {
		return errors.New("request_id is required")
	}
	if strings.TrimSpace(decision) == "" {
		return errors.New("decision is required")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	current, ok := w.decisions[requestID]
	if !ok {
		current = ApprovalDecision{RequestID: requestID, CreatedAt: time.Now().UTC()}
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	current.Decision = strings.TrimSpace(decision)
	current.ApprovedBy = strings.TrimSpace(approvedBy)
	current.Reason = strings.TrimSpace(reason)
	current.CreatedAt = at
	current.ApprovalRequired = current.Decision == "pending"
	w.decisions[requestID] = current
	return nil
}

// Get returns a recorded decision for a request ID if it exists.
func (w *ApprovalWorkflow) Get(requestID string) (ApprovalDecision, bool) {
	if w == nil {
		return ApprovalDecision{}, false
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	decision, ok := w.decisions[requestID]
	return decision, ok
}

// Decisions returns all recorded decisions sorted by timestamp.
func (w *ApprovalWorkflow) Decisions() []ApprovalDecision {
	w.mu.RLock()
	defer w.mu.RUnlock()
	items := make([]ApprovalDecision, 0, len(w.decisions))
	for _, decision := range w.decisions {
		items = append(items, decision)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items
}

// AuditEvent captures a structured audit log entry.
type AuditEvent struct {
	RequestID string    `json:"request_id"`
	EventType string    `json:"event_type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditLogger stores audit events in memory for future investigation.
type AuditLogger struct {
	mu     sync.RWMutex
	events []AuditEvent
}

// NewAuditLogger creates a minimal audit store.
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{events: make([]AuditEvent, 0, 16)}
}

// Record appends an audit record for one request lifecycle event.
func (l *AuditLogger) Record(requestID, eventType, message string) error {
	if strings.TrimSpace(requestID) == "" {
		return errors.New("request_id is required")
	}
	if strings.TrimSpace(eventType) == "" {
		return errors.New("event_type is required")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, AuditEvent{
		RequestID: strings.TrimSpace(requestID),
		EventType: strings.TrimSpace(eventType),
		Message:   strings.TrimSpace(message),
		CreatedAt: time.Now().UTC(),
	})
	return nil
}

// Events returns all recorded events in insertion order.
func (l *AuditLogger) Events() []AuditEvent {
	l.mu.RLock()
	defer l.mu.RUnlock()
	copied := make([]AuditEvent, len(l.events))
	copy(copied, l.events)
	return copied
}
