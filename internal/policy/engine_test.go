package policy

import "testing"

func TestEvaluateRequestAllowsAnalysisOnlyMode(t *testing.T) {
	decision, err := EvaluateRequest(RequestInput{Purpose: "review architecture", ExecutionMode: "analysis_only"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !decision.Allowed {
		t.Fatal("analysis_only should be allowed")
	}
	if decision.ApprovalRequired {
		t.Fatal("analysis_only should not require approval")
	}
}

func TestEvaluateRequestRejectsGovernedExecutionMode(t *testing.T) {
	decision, err := EvaluateRequest(RequestInput{Purpose: "perform mutation", ExecutionMode: "governed"})
	if err == nil {
		t.Fatal("governed execution mode should be rejected")
	}
	if decision.Allowed {
		t.Fatal("governed execution mode should be denied")
	}
	if !decision.ApprovalRequired {
		t.Fatal("governed execution mode should be flagged as approval-required for future workflow")
	}
}

func TestEvaluateRequestRejectsBlankPurpose(t *testing.T) {
	decision, err := EvaluateRequest(RequestInput{Purpose: "   ", ExecutionMode: "analysis_only"})
	if err == nil {
		t.Fatal("blank purpose should be rejected")
	}
	if decision.Allowed {
		t.Fatal("blank purpose should be denied")
	}
}
