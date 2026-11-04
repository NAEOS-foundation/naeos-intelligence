package observability

import "testing"

func TestLoggerRecordsStructuredEvents(t *testing.T) {
	logger := NewLogger("naeos-intelligence")
	entry := logger.WithRequestID("req-123").WithTraceID("trace-456").Info("request_accepted", "status", "accepted")
	if entry.RequestID != "req-123" {
		t.Fatalf("request_id=%q, want req-123", entry.RequestID)
	}
	if entry.TraceID != "trace-456" {
		t.Fatalf("trace_id=%q, want trace-456", entry.TraceID)
	}
	if entry.Event == "" {
		t.Fatal("event must be set")
	}
}

func TestCorrelationContextProducesStableValues(t *testing.T) {
	ctx := NewContext("req-7", "trace-8")
	if ctx.RequestID != "req-7" || ctx.TraceID != "trace-8" {
		t.Fatalf("unexpected correlation context: %+v", ctx)
	}
}
