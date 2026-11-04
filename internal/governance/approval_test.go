package governance

import (
	"testing"
	"time"
)

func TestApprovalWorkflowFlagsGovernedRequests(t *testing.T) {
	approval := NewApprovalWorkflow()
	decision, err := approval.Evaluate("req-123", "governed", "run-mutating-tool")
	if err != nil {
		t.Fatalf("evaluate should not fail: %v", err)
	}
	if !decision.ApprovalRequired {
		t.Fatal("governed requests should require approval")
	}
	if decision.Decision != "pending" {
		t.Fatalf("decision=%q, want pending", decision.Decision)
	}
}

func TestApprovalWorkflowRecordsGrantedDecision(t *testing.T) {
	workflow := NewApprovalWorkflow()
	decision, err := workflow.Evaluate("req-456", "analysis_only", "read-only-review")
	if err != nil {
		t.Fatalf("evaluate should not fail: %v", err)
	}
	if decision.ApprovalRequired {
		t.Fatal("analysis_only requests should not require approval")
	}
	if err := workflow.RecordDecision("req-456", "approved", "ops-user", "review accepted", time.Now()); err != nil {
		t.Fatalf("record decision error: %v", err)
	}
	if len(workflow.Decisions()) != 1 {
		t.Fatalf("decision count=%d, want 1", len(workflow.Decisions()))
	}
}

func TestAuditLogStoresRequestEvents(t *testing.T) {
	logger := NewAuditLogger()
	if err := logger.Record("req-789", "policy_evaluated", "request accepted for analysis_only"); err != nil {
		t.Fatalf("record error: %v", err)
	}
	if len(logger.Events()) != 1 {
		t.Fatalf("event count=%d, want 1", len(logger.Events()))
	}
}
