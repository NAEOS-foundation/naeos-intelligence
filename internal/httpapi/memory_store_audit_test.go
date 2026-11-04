package httpapi

import (
	"context"
	"testing"

	gov "github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
)

func TestMemoryStoreTracksLifecycleAuditEvents(t *testing.T) {
	audit := gov.NewAuditLogger()
	store := NewMemoryStoreWithAudit(audit)
	input := RequestInput{Purpose: "inspect repository", ExecutionMode: "analysis_only"}

	req, replay, err := store.Create(context.Background(), "client-a", "key-1", input, "corr-1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if replay {
		t.Fatal("first create should not be replayed")
	}
	if len(audit.Events()) != 1 {
		t.Fatalf("events=%d, want 1 after create", len(audit.Events()))
	}
	if got := audit.Events()[0].EventType; got != "request_created" {
		t.Fatalf("event=%q, want request_created", got)
	}

	if _, err := store.Cancel(context.Background(), "client-a", req.RequestID, "corr-1"); err != nil {
		t.Fatalf("cancel error: %v", err)
	}
	if len(audit.Events()) != 2 {
		t.Fatalf("events=%d, want 2 after cancel", len(audit.Events()))
	}
	if got := audit.Events()[1].EventType; got != "request_cancelled" {
		t.Fatalf("event=%q, want request_cancelled", got)
	}
}
