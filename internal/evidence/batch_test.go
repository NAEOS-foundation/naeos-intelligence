package evidence

import "testing"

func TestDeterministicEventID_IsStableAndScoped(t *testing.T) {
	base := deterministicEventID("client-a", "00000000-0000-4000-8000-000000000001", "orchestration.completed")
	if got := deterministicEventID("client-a", "00000000-0000-4000-8000-000000000001", "orchestration.completed"); got != base {
		t.Fatalf("same event identity produced different IDs: %q != %q", got, base)
	}
	if got := deterministicEventID("client-b", "00000000-0000-4000-8000-000000000001", "orchestration.completed"); got == base {
		t.Fatal("different client produced the same event ID")
	}
	if got := deterministicEventID("client-a", "00000000-0000-4000-8000-000000000002", "orchestration.completed"); got == base {
		t.Fatal("different request produced the same event ID")
	}
	if got := deterministicEventID("client-a", "00000000-0000-4000-8000-000000000001", "orchestration.planned"); got == base {
		t.Fatal("different event type produced the same event ID")
	}
	if !validUUID(base) {
		t.Fatalf("deterministic event ID is not a valid UUID: %q", base)
	}
}
