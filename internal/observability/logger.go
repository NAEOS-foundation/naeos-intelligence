package observability

import (
	"fmt"
	"strings"
	"time"
)

// CorrelationContext carries stable request and trace identifiers through the service stack.
type CorrelationContext struct {
	RequestID string
	TraceID   string
}

// NewContext creates a structured correlation context for read-only request processing.
func NewContext(requestID, traceID string) CorrelationContext {
	return CorrelationContext{RequestID: strings.TrimSpace(requestID), TraceID: strings.TrimSpace(traceID)}
}

// LogEntry is a minimal structured log entry aligned with the observability specification.
type LogEntry struct {
	Timestamp string
	Level     string
	Service   string
	Component string
	Event     string
	RequestID string
	TraceID   string
	Status    string
}

// Logger stores a service name and optional correlation context for structured logs.
type Logger struct {
	Service   string
	Component string
	RequestID string
	TraceID   string
}

// NewLogger creates a logger instance for a component.
func NewLogger(service string) *Logger {
	return &Logger{Service: strings.TrimSpace(service), Component: "core"}
}

// WithRequestID adds request correlation to this logger instance.
func (l *Logger) WithRequestID(requestID string) *Logger {
	clone := *l
	clone.RequestID = strings.TrimSpace(requestID)
	return &clone
}

// WithTraceID adds trace correlation to this logger instance.
func (l *Logger) WithTraceID(traceID string) *Logger {
	clone := *l
	clone.TraceID = strings.TrimSpace(traceID)
	return &clone
}

// Info records an informational event in a structured JSON-like form.
func (l *Logger) Info(event string, kv ...string) LogEntry {
	return l.log("INFO", event, kv...)
}

func (l *Logger) log(level, event string, kv ...string) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Service:   l.Service,
		Component: l.Component,
		Event:     strings.TrimSpace(event),
		RequestID: l.RequestID,
		TraceID:   l.TraceID,
	}
	if len(kv) >= 2 {
		for i := 0; i+1 < len(kv); i += 2 {
			if strings.EqualFold(kv[i], "status") {
				entry.Status = kv[i+1]
			}
		}
	}
	if entry.Event == "" {
		entry.Event = "event"
	}
	if entry.Service == "" {
		entry.Service = "naeos-intelligence"
	}
	_ = fmt.Sprintf("%v", entry)
	return entry
}
