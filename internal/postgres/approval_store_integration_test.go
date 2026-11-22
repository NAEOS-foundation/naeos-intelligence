package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/httpapi"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/testdb"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRuntimeCannotUseDormantApprovalStore(t *testing.T) {
	if os.Getenv("NAEOS_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set NAEOS_POSTGRES_INTEGRATION=1 to run PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := testdb.Bootstrap(ctx)
	if err != nil {
		t.Fatalf("bootstrap isolated PostgreSQL test database: %v", err)
	}
	if err := Up(ctx, database.MigratorDSN, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	pool, err := OpenPool(ctx, database.RuntimeDSN)
	if err != nil {
		t.Fatalf("open runtime pool: %v", err)
	}
	defer pool.Close()
	adminPool, err := pgxpool.New(ctx, database.AdminDSN)
	if err != nil {
		t.Fatalf("open admin verification pool: %v", err)
	}
	defer adminPool.Close()
	store := New(pool)

	ownerID := "approval-integration-" + time.Now().UTC().Format("20060102150405.000000000")
	request, _, err := store.Create(ctx, ownerID, "", httpapi.RequestInput{
		Purpose: "approval integration action", ExecutionMode: "governed",
	}, "corr-approval-create")
	if err != nil {
		t.Fatalf("create governed request: %v", err)
	}

	if _, err := store.RecordApprovalDecision(ctx, "other-owner", request.RequestID, "approved", "reviewer", "ok"); !errors.Is(err, httpapi.ErrNotFound) {
		t.Fatalf("cross-owner approval decision should be not found, got %v", err)
	}
	if _, err := store.RecordApprovalDecision(ctx, ownerID, request.RequestID, "approved", "reviewer", "approved for analysis"); !errors.Is(err, httpapi.ErrNotFound) {
		t.Fatalf("dormant approval store must be unavailable to the API runtime, got %v", err)
	}
	finalRequest, err := store.Get(ctx, ownerID, request.RequestID)
	if err != nil || finalRequest.Status != httpapi.StatusAccepted {
		t.Fatalf("request status=%s err=%v; denied approval must not change state", finalRequest.Status, err)
	}

	var approvalCount, auditCount, queuedTransitionCount int
	if err := adminPool.QueryRow(ctx, "SELECT count(*) FROM naeos_intelligence.approval_decisions WHERE request_id=$1::uuid", request.RequestID).Scan(&approvalCount); err != nil {
		t.Fatal(err)
	}
	if err := adminPool.QueryRow(ctx, "SELECT count(*) FROM naeos_intelligence.governance_audit_events WHERE request_id=$1::uuid", request.RequestID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if err := adminPool.QueryRow(ctx, "SELECT count(*) FROM naeos_intelligence.request_transitions WHERE request_id=$1::uuid AND to_status='queued'", request.RequestID).Scan(&queuedTransitionCount); err != nil {
		t.Fatal(err)
	}
	if approvalCount != 0 || auditCount != 0 || queuedTransitionCount != 0 {
		t.Fatalf("denied approval left side effects: approvals=%d audit=%d queued_transitions=%d", approvalCount, auditCount, queuedTransitionCount)
	}

	if _, err := pool.Exec(ctx, "UPDATE naeos_intelligence.governance_audit_events SET message='tampered' WHERE request_id=$1::uuid", request.RequestID); !isPermissionDenied(err) {
		t.Fatalf("runtime must not update append-only audit events, got %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM naeos_intelligence.governance_audit_events WHERE request_id=$1::uuid", request.RequestID); !isPermissionDenied(err) {
		t.Fatalf("runtime must not delete append-only audit events, got %v", err)
	}
}
