package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/httpapi"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/testdb"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreLifecycleAndIdempotency(t *testing.T) {
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
		t.Fatalf("open pool: %v", err)
	}
	defer pool.Close()
	store := New(pool)
	clientID := "integration-" + time.Now().UTC().Format("20060102150405.000000000")
	input := httpapi.RequestInput{Purpose: "persistence integration", ExecutionMode: "analysis_only", DesiredOutputs: []string{"trace"}}

	first, replay, err := store.Create(ctx, clientID, "idem-key", input, "correlation-test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if replay || first.Status != httpapi.StatusAccepted {
		t.Fatalf("first create replay=%v status=%s", replay, first.Status)
	}

	// FORCE RLS must hide owner rows when the transaction-local owner context
	// is absent, even though the runtime role has table-level SELECT.
	var unscopedCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM intelligence_requests WHERE id=$1::uuid", first.RequestID).Scan(&unscopedCount); err != nil {
		t.Fatalf("query without owner context: %v", err)
	}
	if unscopedCount != 0 {
		t.Fatalf("unscoped runtime query returned %d rows, want 0", unscopedCount)
	}
	if _, err := pool.Exec(ctx, "SELECT * FROM request_attempts LIMIT 1"); !isPermissionDenied(err) {
		t.Fatalf("runtime must not read worker attempts, got %v", err)
	}
	if _, err := pool.Exec(ctx, "SELECT * FROM approval_decisions LIMIT 1"); !isPermissionDenied(err) {
		t.Fatalf("runtime must not read approval decisions, got %v", err)
	}
	second, replay, err := store.Create(ctx, clientID, "idem-key", input, "correlation-test")
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if !replay || second.RequestID != first.RequestID {
		t.Fatalf("replay=%v id=%s want id=%s", replay, second.RequestID, first.RequestID)
	}
	conflicting := input
	conflicting.Purpose = "changed payload"
	if _, _, err = store.Create(ctx, clientID, "idem-key", conflicting, "correlation-test"); !errors.Is(err, httpapi.ErrIdempotencyConflict) {
		t.Fatalf("want idempotency conflict, got %v", err)
	}
	if _, err = store.Get(ctx, "another-client", first.RequestID); !errors.Is(err, httpapi.ErrNotFound) {
		t.Fatalf("cross-client access err=%v", err)
	}
	cancelled, err := store.Cancel(ctx, clientID, first.RequestID, "correlation-cancel")
	if err != nil || cancelled.Status != httpapi.StatusCancelled {
		t.Fatalf("cancelled=%+v err=%v", cancelled, err)
	}
	if _, err = store.Cancel(ctx, clientID, first.RequestID, "correlation-cancel"); err != nil {
		t.Fatalf("idempotent cancel: %v", err)
	}

	// Concurrent cancellation requests must serialize against the row lock and
	// append exactly one lifecycle transition while remaining idempotent.
	raceRequest, _, err := store.Create(ctx, clientID+"-race", "", input, "correlation-race")
	if err != nil {
		t.Fatalf("create cancellation-race request: %v", err)
	}
	const cancelCallCount = 12
	var cancelWG sync.WaitGroup
	cancelErrors := make(chan error, cancelCallCount)
	for i := 0; i < cancelCallCount; i++ {
		cancelWG.Add(1)
		go func() {
			defer cancelWG.Done()
			result, cancelErr := store.Cancel(ctx, clientID+"-race", raceRequest.RequestID, "correlation-race-cancel")
			if cancelErr != nil {
				cancelErrors <- cancelErr
				return
			}
			if result.Status != httpapi.StatusCancelled {
				cancelErrors <- errors.New("concurrent cancel returned non-cancelled status")
			}
		}()
	}
	cancelWG.Wait()
	close(cancelErrors)
	for cancelErr := range cancelErrors {
		t.Errorf("concurrent cancellation: %v", cancelErr)
	}
	adminPool, err := pgxpool.New(ctx, database.AdminDSN)
	if err != nil {
		t.Fatalf("open admin verification pool: %v", err)
	}
	defer adminPool.Close()
	var publicRequestAbsent, schemaRequestPresent, publicLedgerAbsent, schemaLedgerPresent bool
	if err := adminPool.QueryRow(ctx, `
		SELECT to_regclass('public.intelligence_requests') IS NULL,
		       to_regclass('naeos_intelligence.intelligence_requests') IS NOT NULL,
		       to_regclass('public.goose_db_version') IS NULL,
		       to_regclass('naeos_intelligence.goose_db_version') IS NOT NULL
	`).Scan(&publicRequestAbsent, &schemaRequestPresent, &publicLedgerAbsent, &schemaLedgerPresent); err != nil {
		t.Fatalf("inspect schema placement: %v", err)
	}
	if !publicRequestAbsent || !schemaRequestPresent || !publicLedgerAbsent || !schemaLedgerPresent {
		t.Fatalf("schema isolation failed: public request absent=%v schema request present=%v public ledger absent=%v schema ledger present=%v", publicRequestAbsent, schemaRequestPresent, publicLedgerAbsent, schemaLedgerPresent)
	}
	for _, query := range []string{
		"SELECT * FROM public.crm_guard_test",
		"SELECT * FROM naeos_control_plane.guard_test",
		"SELECT * FROM naeos_identity.guard_test",
		"SELECT * FROM naeos_intelligence.goose_db_version",
		"CREATE TABLE naeos_intelligence.runtime_forbidden (id integer)",
	} {
		if _, err := pool.Exec(ctx, query); !isPermissionDenied(err) {
			t.Fatalf("runtime isolation query %q should be denied with SQLSTATE 42501, got %v", query, err)
		}
	}
	var transitionCount int
	if err := adminPool.QueryRow(ctx, "SELECT count(*) FROM naeos_intelligence.request_transitions WHERE request_id=$1::uuid AND event_type='request.cancelled'", raceRequest.RequestID).Scan(&transitionCount); err != nil {
		t.Fatalf("count cancellation transitions: %v", err)
	}
	if transitionCount != 1 {
		t.Fatalf("concurrent cancel created %d transitions, want exactly 1", transitionCount)
	}
	finalRequest, err := store.Get(ctx, clientID+"-race", raceRequest.RequestID)
	if err != nil || finalRequest.Status != httpapi.StatusCancelled {
		t.Fatalf("final race request status=%s err=%v", finalRequest.Status, err)
	}

	if _, err := store.CreateAttempt(ctx, first.RequestID); !errors.Is(err, httpapi.ErrNotFound) {
		t.Fatalf("runtime attempt creation must fail closed without owner/worker context, got %v", err)
	}
	if err := store.StartAttempt(ctx, first.RequestID, 1, "worker-integration"); !isPermissionDenied(err) {
		t.Fatalf("runtime must not update worker attempts, got %v", err)
	}
	if err := store.FinishAttempt(ctx, first.RequestID, 1, "failed", "transient", "simulated integration failure"); !isPermissionDenied(err) {
		t.Fatalf("runtime must not finish worker attempts, got %v", err)
	}
}

func isPermissionDenied(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42501"
}
