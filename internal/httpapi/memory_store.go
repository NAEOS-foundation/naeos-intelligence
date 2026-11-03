package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

type idempotencyRecord struct {
	digest    string
	requestID string
}

type MemoryStore struct {
	mu          sync.RWMutex
	requests    map[string]Request
	idempotency map[string]idempotencyRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{requests: make(map[string]Request), idempotency: make(map[string]idempotencyRecord)}
}

func payloadDigest(input RequestInput) (string, error) {
	b, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func generatedID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *MemoryStore) Create(clientID, key string, input RequestInput) (Request, bool, error) {
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
			return req, true, nil
		}
	}
	id, err := generatedID()
	if err != nil {
		return Request{}, false, err
	}
	now := time.Now().UTC()
	req := Request{RequestID: id, Status: StatusAccepted, Purpose: input.Purpose, ExecutionMode: input.ExecutionMode, CreatedAt: now, UpdatedAt: now, ClientID: clientID, Input: input}
	s.requests[req.RequestID] = req
	if key != "" {
		s.idempotency[idemKey] = idempotencyRecord{digest: digest, requestID: req.RequestID}
	}
	return req, false, nil
}

func (s *MemoryStore) Get(clientID, requestID string) (Request, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.requests[requestID]
	if !ok || req.ClientID != clientID {
		return Request{}, ErrNotFound
	}
	return req, nil
}

func (s *MemoryStore) Cancel(clientID, requestID string) (Request, error) {
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
		return req, nil
	case StatusCancelled:
		return req, nil
	default:
		return Request{}, ErrConflict
	}
}
