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
		{"valid analysis_only with nil pool", "req-1", "analysis_only", "read", true},
		{"valid governed with nil pool", "req-2", "governed", "mutate", true},
		{"unsupported mode", "req-3", "execute", "run", true},
		{"empty mode defaults but nil pool", "req-4", "", "read", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		{"valid approved but nil pool", "req-1", "approved", "user", true},
		{"valid rejected but nil pool", "req-2", "rejected", "user", true},
		{"invalid decision", "req-3", "unexpected", "user", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestValidateDecisionEnum(t *testing.T) {
	for _, decision := range []string{"pending", "approved", "rejected"} {
		if err := validateDecision(decision); err != nil {
			t.Errorf("validateDecision(%q) unexpected error: %v", decision, err)
		}
	}
	for _, decision := range []string{"", "denied", "unknown", "APPROVED"} {
		if err := validateDecision(decision); err == nil {
			t.Errorf("validateDecision(%q) expected error", decision)
		}
	}
}

func TestNilPoolGuards(t *testing.T) {
	var nilStore *Store
	store := New(nil)
	ctx := context.Background()
	for name, s := range map[string]*Store{"nil receiver": nilStore, "nil pool": store} {
		t.Run(name, func(t *testing.T) {
			if err := s.Ping(ctx); err == nil { t.Fatal("Ping() expected error") }
			if _, err := s.Evaluate(ctx, "req-1", "analysis_only", "read"); err == nil { t.Fatal("Evaluate() expected error") }
			if err := s.RecordDecision(ctx, "req-1", "approved", "user", "reason", time.Time{}); err == nil { t.Fatal("RecordDecision() expected error") }
			if _, ok := s.Get(ctx, "req-1"); ok { t.Fatal("Get() expected ok=false") }
			if _, err := s.Decisions(ctx); err == nil { t.Fatal("Decisions() expected error") }
			s.Close()
		})
	}
}
