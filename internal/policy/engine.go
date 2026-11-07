package policy

import (
	"fmt"
	"strings"
)

// RequestInput is the minimal contract required by the governance boundary.
type RequestInput struct {
	Purpose       string
	ExecutionMode string
}

// Decision captures whether a request may proceed and whether an approval step
// is required in a future governed workflow.
type Decision struct {
	Allowed          bool   `json:"allowed"`
	ApprovalRequired bool   `json:"approval_required"`
	Reason           string `json:"reason,omitempty"`
}

// EvaluateRequest enforces the current read-only MVP boundary: only analysis_only
// requests are allowed. Governed execution is intentionally rejected and marked
// for a future approved workflow.
func EvaluateRequest(in RequestInput) (Decision, error) {
	trimmedPurpose := strings.TrimSpace(in.Purpose)
	if trimmedPurpose == "" || len(trimmedPurpose) > 2000 {
		return Decision{Allowed: false, ApprovalRequired: false, Reason: "purpose must contain 1 to 2000 characters"}, fmt.Errorf("purpose must contain 1 to 2000 characters")
	}

	switch in.ExecutionMode {
	case "analysis_only":
		return Decision{Allowed: true, ApprovalRequired: false, Reason: "analysis-only requests are permitted in the read-only MVP"}, nil
	case "governed":
		return Decision{Allowed: false, ApprovalRequired: true, Reason: "governed execution requires a separately approved approval and runtime policy flow"}, fmt.Errorf("execution_mode must be analysis_only in the MVP")
	default:
		return Decision{Allowed: false, ApprovalRequired: false, Reason: "execution_mode must be analysis_only in the MVP"}, fmt.Errorf("execution_mode must be analysis_only in the MVP")
	}
}
