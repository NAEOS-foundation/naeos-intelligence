package evidence_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/evidence"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/postgres"
)

func TestRecordEventsPostgresAtomicReplayAndConflict(t *testing.T) {
	if os.Getenv("NAEOS_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set NAEOS_POSTGRES_INTEGRATION=1 to run PostgreSQL evidence integration test")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL must be set for PostgreSQL evidence integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := postgres.Up(ctx, dsn, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	pool, err := postgres.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL pool: %v", err)
	}
	defer pool.Close()

	store := evidence.New(pool)
	requestID := fmt.Sprintf("00000000-0000-4000-8000-%012x", uint64(time.Now().UnixNano())&0xffffffffffff)
	clientID := fmt.Sprintf("evidence-integration-%d", time.Now().UnixNano())
	batch := []evidence.Event{
		{EventType: "integration.classified", ActorType: "system", ActorID: "test", Message: "intent=repository_analysis"},
		{EventType: "integration.planned", ActorType: "system", ActorID: "test", Message: "tasks=1"},
	}

	if err := store.RecordEvents(ctx, clientID, requestID, batch); err != nil {
		t.Fatalf("initial batch: %v", err)
	}
	if err := store.RecordEvents(ctx, clientID, requestID, batch); err != nil {
		t.Fatalf("exact replay: %v", err)
	}
	events, err := store.GetEvents(ctx, clientID, requestID)
	if err != nil {
		t.Fatalf("read after replay: %v", err)
	}
	if len(events) != len(batch) {
		t.Fatalf("event count after replay = %d, want %d", len(events), len(batch))
	}

	// The first event is new, but the second collides with an existing
	// deterministic identity and has different content. The whole transaction
	// must roll back, including the first insert.
	conflicting := []evidence.Event{
		{EventType: "integration.rollback_probe", ActorType: "system", ActorID: "test", Message: "must roll back"},
		{EventType: "integration.planned", ActorType: "system", ActorID: "test", Message: "tampered content"},
	}
	if err := store.RecordEvents(ctx, clientID, requestID, conflicting); err == nil {
		t.Fatal("conflicting replay unexpectedly succeeded")
	}
	events, err = store.GetEvents(ctx, clientID, requestID)
	if err != nil {
		t.Fatalf("read after conflict: %v", err)
	}
	if len(events) != len(batch) {
		t.Fatalf("event count after failed transaction = %d, want %d", len(events), len(batch))
	}
	for _, event := range events {
		if event.EventType == "integration.rollback_probe" {
			t.Fatal("earlier insert survived failed batch; transaction was not atomic")
		}
	}
}
