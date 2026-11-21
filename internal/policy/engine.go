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

// Decision captures admission and approval requirements. Approval is not
// execution authorization; runtime policy must independently authorize actions.
type Decision struct {
	Allowed          bool   `json:"allowed"`
	ApprovalRequired bool   `json:"approval_required"`
	Reason           string `json:"reason,omitempty"`
}

// EvaluateRequest admits analysis-only requests and stages governed requests
// behind an explicit human approval boundary. Governed requests remain denied
// for execution until a separate authorization decision is made at runtime.
func EvaluateRequest(in RequestInput) (Decision, error) {
	trimmedPurpose := strings.TrimSpace(in.Purpose)
	if trimmedPurpose == "" || len(trimmedPurpose) > 2000 {
		return Decision{Allowed: false, ApprovalRequired: false, Reason: "purpose must contain 1 to 2000 characters"}, fmt.Errorf("purpose must contain 1 to 2000 characters")
	}

	switch in.ExecutionMode {
	case "analysis_only":
		return Decision{Allowed: true, ApprovalRequired: false, Reason: "analysis-only requests are permitted in the read-only MVP"}, nil
	case "governed":
		return Decision{Allowed: false, ApprovalRequired: true, Reason: "governed request requires explicit human approval before authorization"}, nil
	default:
		return Decision{Allowed: false, ApprovalRequired: false, Reason: "execution_mode must be analysis_only or governed"}, fmt.Errorf("execution_mode must be analysis_only or governed")
	}
}
