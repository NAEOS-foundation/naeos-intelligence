package httpapi

import (
	"errors"
	"time"
)

type Status string

const (
	StatusAccepted Status = "accepted"
	StatusQueued Status = "queued"
	StatusRunning Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed Status = "failed"
	StatusCancelled Status = "cancelled"
	StatusRequiresApproval Status = "requires_approval"
)

var ErrNotFound = errors.New("request not found")
var ErrConflict = errors.New("request state conflict")
var ErrIdempotencyConflict = errors.New("idempotency key payload conflict")

type ContextSource struct { Type string `json:"type"`; Reference string `json:"reference"` }

type RequestInput struct {
	Purpose string `json:"purpose"`
	ContextSources []ContextSource `json:"context_sources,omitempty"`
	ExecutionMode string `json:"execution_mode"`
	DesiredOutputs []string `json:"desired_outputs,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Request struct {
	RequestID string `json:"request_id"`
	Status Status `json:"status"`
	Purpose string `json:"purpose"`
	ExecutionMode string `json:"execution_mode"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ClientID string `json:"-"`
	Input RequestInput `json:"-"`
}

type Accepted struct {
	RequestID string `json:"request_id"`
	Status Status `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	StatusURL string `json:"status_url"`
}

type CancellationAccepted struct {
	RequestID string `json:"request_id"`
	Status Status `json:"status"`
	Message string `json:"message"`
}

type Store interface {
	Create(clientID, idempotencyKey string, input RequestInput) (Request, bool, error)
	Get(clientID, requestID string) (Request, error)
	Cancel(clientID, requestID string) (Request, error)
}
