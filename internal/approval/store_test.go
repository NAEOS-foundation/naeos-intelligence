package approval

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
)

func TestNewUUIDFormat(t *testing.T) {
	id, err := newUUID()
	if err != nil {
		t.Fatalf("newUUID() error = %v", err)
	}
	if len(id) != 36 {
		t.Errorf("newUUID() = %q, expected UUID format length 36", id)
	}
	// Two calls should produce different IDs.
	id2, err := newUUID()
	if err != nil {
		t.Fatalf("newUUID() error = %v", err)
	}
	if id == id2 {
		t.Error("newUUID() produced duplicate IDs")
	}
}

func TestNullIfBlank(t *testing.T) {
	if nullIfBlank("") != nil {
		t.Error("nullIfBlank(\"\") = non-nil, want nil")
	}
	if nullIfBlank("   ") != nil {
		t.Error("nullIfBlank(\"   \") = non-nil, want nil")
	}
	v := nullIfBlank("abc")
	if v == nil || v.(string) != "abc" {
		t.Errorf("nullIfBlank(\"abc\") = %v, want \"abc\"", v)
	}
}

func TestEvaluateValidation(t *testing.T) {
	s := New(nil)

	tests := []struct {
		name          string
		requestID     string
		executionMode string
		action        string
		wantErr       bool
	}{
		{"empty request ID", "", "analysis_only", "read", true},
		{"empty action", "req-1", "analysis_only", "", true},
		{"valid analysis_only", "req-1", "analysis_only", "read", false},
		{"valid governed", "req-2", "governed", "mutate", false},
		{"unsupported mode", "req-3", "execute", "run", true},
		{"empty mode defaults to analysis_only", "req-4", "", "read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validation-only checks: if pool is nil, only validation errors
			// are reachable. Valid inputs that pass validation will try DB
			// access and panic on nil pool, so we skip those.
			if s.pool == nil && !tt.wantErr {
				t.Skip("skipping DB-dependent test with nil pool")
			}
			_, err := s.Evaluate(context.Background(), tt.requestID, tt.executionMode, tt.action)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Evaluate() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
		})
	}
}

func TestRecordDecisionValidation(t *testing.T) {
	s := New(nil)

	tests := []struct {
		name       string
		requestID  string
		decision   string
		approvedBy string
		wantErr    bool
	}{
		{"empty request ID", "", "approved", "user", true},
		{"empty decision", "req-1", "", "user", true},
		{"valid", "req-1", "approved", "user", false},
		{"valid denied", "req-2", "denied", "user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if s.pool == nil && !tt.wantErr {
				t.Skip("skipping DB-dependent test with nil pool")
			}
			err := s.RecordDecision(context.Background(), tt.requestID, tt.decision, tt.approvedBy, "reason", time.Time{})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("RecordDecision() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("RecordDecision() unexpected error: %v", err)
			}
		})
	}
}

func TestGetAndDecisionsValidation(t *testing.T) {
	s := New(nil)

	// Get with empty request ID should return false, not error
	d, ok := s.Get(context.Background(), "")
	if ok {
		t.Error("Get with empty request ID should return ok=false")
	}
	if d.RequestID != "" {
		t.Error("Get with empty request ID should return empty decision")
	}

	// Decisions with nil pool - expect DB error or nil, not panic
	if s.pool != nil {
		_, err := s.Decisions(context.Background())
		if err != nil && !strings.Contains(err.Error(), "no connection") && !strings.Contains(err.Error(), "closed") {
			t.Fatalf("Decisions() unexpected error: %v", err)
		}
	}
}

// Ensure governance.ApprovalDecision type is used
var _ = governance.ApprovalDecision{}
