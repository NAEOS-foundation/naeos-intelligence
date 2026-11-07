package evidence

import (
	"strings"
	"testing"
)

func TestNewUUIDFormat(t *testing.T) {
	id, err := newUUID()
	if err != nil {
		t.Fatalf("newUUID() error = %v", err)
	}
	if !validUUID(id) {
		t.Errorf("newUUID() = %q, not a valid UUID", id)
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

func TestRecordEventValidation(t *testing.T) {
	s := New(nil)

	tests := []struct {
		name      string
		clientID  string
		requestID string
		eventType string
		actorType string
		actorID   string
		message   string
		wantErr   bool
	}{
		{
			name:     "empty client ID",
			clientID: "",
			wantErr:  true,
		},
		{
			name:     "client ID too long",
			clientID: strings.Repeat("a", 256),
			wantErr:  true,
		},
		{
			name:      "invalid request ID",
			clientID:  "client-a",
			requestID: "not-a-uuid",
			wantErr:   true,
		},
		{
			name:      "empty event type",
			clientID:  "client-a",
			requestID: "00000000-0000-0000-0000-000000000000",
			eventType: "",
			wantErr:   true,
		},
		{
			name:      "event type too long",
			clientID:  "client-a",
			requestID: "00000000-0000-0000-0000-000000000000",
			eventType: strings.Repeat("a", 129),
			wantErr:   true,
		},
		{
			name:      "empty actor type",
			clientID:  "client-a",
			requestID: "00000000-0000-0000-0000-000000000000",
			eventType: "request.received",
			actorType: "",
			wantErr:   true,
		},
		{
			name:      "empty actor ID",
			clientID:  "client-a",
			requestID: "00000000-0000-0000-0000-000000000000",
			eventType: "request.received",
			actorType: "system",
			actorID:   "",
			wantErr:   true,
		},
		{
			name:      "empty message",
			clientID:  "client-a",
			requestID: "00000000-0000-0000-0000-000000000000",
			eventType: "request.received",
			actorType: "system",
			actorID:   "sys-1",
			message:   "",
			wantErr:   true,
		},
		{
			name:      "message too long",
			clientID:  "client-a",
			requestID: "00000000-0000-0000-0000-000000000000",
			eventType: "request.received",
			actorType: "system",
			actorID:   "sys-1",
			message:   strings.Repeat("a", 4001),
			wantErr:   true,
		},
		{
			name:      "valid event",
			clientID:  "client-a",
			requestID: "00000000-0000-0000-0000-000000000000",
			eventType: "request.received",
			actorType: "system",
			actorID:   "sys-1",
			message:   "request accepted",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// RecordEvent with a nil pool panics after validation passes.
			// Use recover to distinguish validation errors from DB errors.
			didPanic := false
			var panicVal any
			func() {
				defer func() {
					if r := recover(); r != nil {
						didPanic = true
						panicVal = r
					}
				}()
				err := s.RecordEvent(nil, tt.clientID, tt.requestID, tt.eventType, tt.actorType, tt.actorID, tt.message, "")
				if tt.wantErr {
					if err == nil {
						t.Fatalf("RecordEvent() error = nil, want error")
					}
					return
				}
				if err != nil && !isValidationError(err) {
					t.Fatalf("RecordEvent() unexpected error: %v", err)
				}
			}()
			if tt.wantErr && didPanic {
				t.Fatalf("RecordEvent() panicked with %v, want validation error", panicVal)
			}
			if !tt.wantErr && !didPanic {
				// Validation passed and no panic means the call succeeded without a DB.
				// With nil pool this shouldn't happen; treat as unexpected.
				t.Fatalf("RecordEvent() did not panic with nil pool, unexpected")
			}
		})
	}
}

func isValidationError(err error) bool {
	return err != nil &&
		(err.Error() == "client ID must contain 1 to 255 characters" ||
			err.Error() == "request ID must be a valid UUID" ||
			err.Error() == "event type must contain 1 to 128 characters" ||
			err.Error() == "actor type must contain 1 to 64 characters" ||
			err.Error() == "actor ID must contain 1 to 255 characters" ||
			err.Error() == "message must contain 1 to 4000 characters" ||
			err.Error() == "correlation ID must not exceed 255 characters")
}
