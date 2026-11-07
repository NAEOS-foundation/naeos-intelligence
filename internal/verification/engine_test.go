package verification

import (
	"context"
	"testing"
)

func TestNewReturnsNonNil(t *testing.T) {
	e := New()
	if e == nil {
		t.Fatal("New() returned nil engine")
	}
}

func TestVerify_MatchingOutcomes(t *testing.T) {
	e := New()
	rec, err := e.Verify(context.Background(), "exec-1", "completed", "completed")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if rec.Result != ResultVerified {
		t.Errorf("result = %q, want %q", rec.Result, ResultVerified)
	}
	if rec.ExecutionID != "exec-1" {
		t.Errorf("execution_id = %q, want exec-1", rec.ExecutionID)
	}
	if rec.Method != "outcome_comparison" {
		t.Errorf("method = %q, want outcome_comparison", rec.Method)
	}
	if rec.VerificationID == "" {
		t.Error("verification_id is empty")
	}
	if rec.VerifiedAt.IsZero() {
		t.Error("verified_at is zero")
	}
}

func TestVerify_DifferentOutcomes(t *testing.T) {
	e := New()
	rec, err := e.Verify(context.Background(), "exec-1", "completed", "failed")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if rec.Result != ResultFailed {
		t.Errorf("result = %q, want %q", rec.Result, ResultFailed)
	}
}

func TestVerify_EmptyObservedOutcome(t *testing.T) {
	e := New()
	rec, err := e.Verify(context.Background(), "exec-1", "completed", "")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if rec.Result != ResultInconclusive {
		t.Errorf("result = %q, want %q", rec.Result, ResultInconclusive)
	}
}

func TestVerify_EmptyExecutionID(t *testing.T) {
	e := New()
	_, err := e.Verify(context.Background(), "", "completed", "completed")
	if err == nil {
		t.Fatal("Verify() error = nil, want error")
	}
	if err.Error() != "execution ID is required" {
		t.Errorf("error = %q, want execution ID is required", err.Error())
	}
}

func TestVerify_ContextCancelled(t *testing.T) {
	e := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := e.Verify(ctx, "exec-1", "completed", "completed")
	if err == nil {
		t.Fatal("Verify() error = nil, want context cancelled error")
	}
}

func TestVerifyIDsAreUnique(t *testing.T) {
	e := New()
	r1, err := e.Verify(context.Background(), "exec-1", "completed", "completed")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	r2, err := e.Verify(context.Background(), "exec-2", "completed", "completed")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if r1.VerificationID == r2.VerificationID {
		t.Error("verification IDs are not unique")
	}
}
