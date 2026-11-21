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

func TestEvaluateRequestStagesGovernedModeForApproval(t *testing.T) {
	decision, err := EvaluateRequest(RequestInput{Purpose: "perform mutation", ExecutionMode: "governed"})
	if err != nil {
		t.Fatalf("governed request should be accepted for approval staging: %v", err)
	}
	if decision.Allowed {
		t.Fatal("governed request must not be authorized for execution")
	}
	if !decision.ApprovalRequired {
		t.Fatal("governed request must require explicit approval")
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
