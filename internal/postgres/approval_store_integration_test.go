package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/httpapi"
)

func TestPostgresDurableApprovalDecision(t *testing.T) {
	if os.Getenv("NAEOS_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set NAEOS_POSTGRES_INTEGRATION=1 to run PostgreSQL integration test")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL must be set for PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := Up(ctx, dsn, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer pool.Close()
	store := New(pool)

	clientID := "approval-integration-" + time.Now().UTC().Format("20060102150405.000000000")
	request, _, err := store.Create(ctx, clientID, "", httpapi.RequestInput{
		Purpose: "approval integration action", ExecutionMode: "governed",
	}, "corr-approval-create")
	if err != nil {
		t.Fatalf("create governed request: %v", err)
	}

	if _, err := store.RecordApprovalDecision(ctx, "other-tenant", request.RequestID, "approved", "reviewer", "ok"); !errors.Is(err, httpapi.ErrNotFound) {
		t.Fatalf("cross-tenant decision should be not found, got %v", err)
	}
	decision, err := store.RecordApprovalDecision(ctx, clientID, request.RequestID, "approved", "reviewer", "approved for analysis")
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}
	if decision.Decision != "approved" || decision.ApprovedBy != "reviewer" || decision.ApprovalRequired {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	finalRequest, err := store.Get(ctx, clientID, request.RequestID)
	if err != nil || finalRequest.Status != httpapi.StatusQueued {
		t.Fatalf("approved request status=%s err=%v, want queued", finalRequest.Status, err)
	}

	replay, err := store.RecordApprovalDecision(ctx, clientID, request.RequestID, "approved", "reviewer", "approved for analysis")
	if err != nil || replay.Decision != "approved" {
		t.Fatalf("identical decision replay=%+v err=%v", replay, err)
	}
	if _, err := store.RecordApprovalDecision(ctx, clientID, request.RequestID, "denied", "reviewer", "changed"); !errors.Is(err, httpapi.ErrConflict) {
		t.Fatalf("conflicting duplicate should be conflict, got %v", err)
	}

	var approvalCount, auditCount, transitionCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM approval_decisions WHERE request_id=$1::uuid", request.RequestID).Scan(&approvalCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM governance_audit_events WHERE request_id=$1::uuid", request.RequestID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM request_transitions WHERE request_id=$1::uuid AND to_status='queued'", request.RequestID).Scan(&transitionCount); err != nil {
		t.Fatal(err)
	}
	if approvalCount != 1 || auditCount != 1 || transitionCount != 1 {
		t.Fatalf("persisted counts approval=%d audit=%d transitions=%d; want all 1", approvalCount, auditCount, transitionCount)
	}

	if _, err := pool.Exec(ctx, "UPDATE governance_audit_events SET message='tampered' WHERE request_id=$1::uuid", request.RequestID); err == nil {
		t.Fatal("audit update unexpectedly succeeded")
	}
	if _, err := pool.Exec(ctx, "DELETE FROM governance_audit_events WHERE request_id=$1::uuid", request.RequestID); err == nil {
		t.Fatal("audit delete unexpectedly succeeded")
	}
}
