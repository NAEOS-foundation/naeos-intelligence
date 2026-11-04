package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	gov "github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
)

type idempotencyRecord struct {
	digest    string
	requestID string
}

type MemoryStore struct {
	mu          sync.RWMutex
	requests    map[string]Request
	idempotency map[string]idempotencyRecord
	audit       *gov.AuditLogger
	approval    *gov.ApprovalWorkflow
}

func NewMemoryStore() *MemoryStore {
	return NewMemoryStoreWithApproval(gov.NewApprovalWorkflow())
}

func NewMemoryStoreWithAudit(audit *gov.AuditLogger) *MemoryStore {
	if audit == nil {
		audit = gov.NewAuditLogger()
	}
	return NewMemoryStoreWithApprovalAndAudit(gov.NewApprovalWorkflow(), audit)
}

func NewMemoryStoreWithApproval(approval *gov.ApprovalWorkflow) *MemoryStore {
	return NewMemoryStoreWithApprovalAndAudit(approval, gov.NewAuditLogger())
}

func NewMemoryStoreWithApprovalAndAudit(approval *gov.ApprovalWorkflow, audit *gov.AuditLogger) *MemoryStore {
	if approval == nil {
		approval = gov.NewApprovalWorkflow()
	}
	if audit == nil {
		audit = gov.NewAuditLogger()
	}
	return &MemoryStore{requests: make(map[string]Request), idempotency: make(map[string]idempotencyRecord), audit: audit, approval: approval}
}

func payloadDigest(input RequestInput) (string, error) {
	b, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func requestStatusForInput(input RequestInput) (Status, bool) {
	if strings.EqualFold(strings.TrimSpace(input.ExecutionMode), "governed") {
		return StatusRequiresApproval, true
	}
	return StatusAccepted, false
}

func generatedID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *MemoryStore) Create(ctx context.Context, clientID, key string, input RequestInput, correlationID string) (Request, bool, error) {
	if err := ctx.Err(); err != nil {
		return Request{}, false, err
	}
	digest, err := payloadDigest(input)
	if err != nil {
		return Request{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idemKey := clientID + "\x00" + key
	if key != "" {
		if previous, ok := s.idempotency[idemKey]; ok {
			if previous.digest != digest {
				return Request{}, false, ErrIdempotencyConflict
			}
			req, exists := s.requests[previous.requestID]
			if !exists {
				return Request{}, false, ErrNotFound
			}
			if err := s.audit.Record(req.RequestID, "request_replayed", fmt.Sprintf("client %s replayed request with idempotency key", clientID)); err != nil {
				return Request{}, false, err
			}
			return req, true, nil
		}
	}
	id, err := generatedID()
	if err != nil {
		return Request{}, false, err
	}
	now := time.Now().UTC()
	status, approvalRequired := requestStatusForInput(input)
	req := Request{RequestID: id, Status: status, ApprovalRequired: approvalRequired, Purpose: input.Purpose, ExecutionMode: input.ExecutionMode, CreatedAt: now, UpdatedAt: now, ClientID: clientID, Input: input}
	s.requests[req.RequestID] = req
	if key != "" {
		s.idempotency[idemKey] = idempotencyRecord{digest: digest, requestID: req.RequestID}
	}
	if status == StatusRequiresApproval && s.approval != nil {
		if _, err := s.approval.Evaluate(req.RequestID, input.ExecutionMode, input.Purpose); err != nil {
			return Request{}, false, err
		}
		if err := s.audit.Record(req.RequestID, "approval_required", fmt.Sprintf("request %s requires explicit approval before execution", req.RequestID)); err != nil {
			return Request{}, false, err
		}
	}
	if err := s.audit.Record(req.RequestID, "request_created", fmt.Sprintf("client %s submitted request", clientID)); err != nil {
		return Request{}, false, err
	}
	return req, false, nil
}

func (s *MemoryStore) ApprovalDecision(requestID string) (gov.ApprovalDecision, bool) {
	if s == nil || s.approval == nil {
		return gov.ApprovalDecision{}, false
	}
	return s.approval.Get(requestID)
}

func (s *MemoryStore) RecordApprovalDecision(ctx context.Context, clientID, requestID, decision, approvedBy, reason string) (gov.ApprovalDecision, error) {
	if s == nil {
		return gov.ApprovalDecision{}, ErrNotFound
	}
	if err := ctx.Err(); err != nil {
		return gov.ApprovalDecision{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return gov.ApprovalDecision{}, err
	}
	// Ownership validation and approval mutation share the store lock so a
	// foreign tenant cannot race the check and change approval state.
	req, ok := s.requests[requestID]
	if !ok || req.ClientID != clientID {
		return gov.ApprovalDecision{}, ErrNotFound
	}
	if s.approval == nil {
		s.approval = gov.NewApprovalWorkflow()
	}
	if err := s.approval.RecordDecision(requestID, strings.TrimSpace(decision), strings.TrimSpace(approvedBy), strings.TrimSpace(reason), time.Now().UTC()); err != nil {
		return gov.ApprovalDecision{}, err
	}
	decisionRecord, ok := s.approval.Get(requestID)
	if !ok {
		return gov.ApprovalDecision{}, ErrNotFound
	}
	switch strings.ToLower(decisionRecord.Decision) {
	case "approved":
		if req.Status == StatusRequiresApproval {
			req.Status = StatusQueued
			req.UpdatedAt = time.Now().UTC()
			s.requests[requestID] = req
		}
	case "denied", "rejected":
		if req.Status == StatusRequiresApproval || req.Status == StatusQueued {
			req.Status = StatusCancelled
			req.UpdatedAt = time.Now().UTC()
			s.requests[requestID] = req
		}
	}
	if err := s.audit.Record(req.RequestID, "approval_decision_recorded", fmt.Sprintf("approval decision=%s for request %s", decisionRecord.Decision, requestID)); err != nil {
		return gov.ApprovalDecision{}, err
	}
	return decisionRecord, nil
}

func (s *MemoryStore) Get(ctx context.Context, clientID, requestID string) (Request, error) {
	if err := ctx.Err(); err != nil {
		return Request{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.requests[requestID]
	if !ok || req.ClientID != clientID {
		return Request{}, ErrNotFound
	}
	return req, nil
}

func (s *MemoryStore) Cancel(ctx context.Context, clientID, requestID, correlationID string) (Request, error) {
	if err := ctx.Err(); err != nil {
		return Request{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.requests[requestID]
	if !ok || req.ClientID != clientID {
		return Request{}, ErrNotFound
	}
	switch req.Status {
	case StatusAccepted, StatusQueued, StatusRequiresApproval:
		req.Status = StatusCancelled
		req.UpdatedAt = time.Now().UTC()
		s.requests[requestID] = req
		if err := s.audit.Record(req.RequestID, "request_cancelled", fmt.Sprintf("client %s cancelled request", clientID)); err != nil {
			return Request{}, err
		}
		return req, nil
	case StatusCancelled:
		if err := s.audit.Record(req.RequestID, "request_cancelled", fmt.Sprintf("client %s confirmed request already cancelled", clientID)); err != nil {
			return Request{}, err
		}
		return req, nil
	default:
		return Request{}, ErrConflict
	}
}
