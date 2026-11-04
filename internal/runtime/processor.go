package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/policy"
)

// Result is the bounded decision produced by the runtime gate.
type Result struct {
	RequestID        string
	ProviderID       string
	Allowed          bool
	ApprovalRequired bool
	Reason           string
}

// Processor is the runtime boundary that validates a request and selects a compatible provider.
type Processor struct {
	Registry         *model.Registry
	ApprovalWorkflow *governance.ApprovalWorkflow
}

// NewProcessor creates a safe runtime processor for the read-only MVP.
func NewProcessor(registry *model.Registry) *Processor {
	if registry == nil {
		registry = model.NewRegistry()
	}
	return &Processor{Registry: registry}
}

// Process evaluates the request against policy and provider capability constraints.
func (p *Processor) Process(requestID, purpose, executionMode string, requiredCapabilities []string) (Result, error) {
	cleanRequestID := strings.TrimSpace(requestID)
	if cleanRequestID == "" {
		return Result{}, fmt.Errorf("request_id is required")
	}
	decision, err := policy.EvaluateRequest(policy.RequestInput{Purpose: purpose, ExecutionMode: executionMode})
	if p != nil && p.ApprovalWorkflow != nil {
		if approvalDecision, approvalErr := p.ApprovalWorkflow.Evaluate(cleanRequestID, executionMode, purpose); approvalErr == nil && approvalDecision.ApprovalRequired {
			if err != nil {
				return Result{RequestID: cleanRequestID, Allowed: false, ApprovalRequired: true, Reason: approvalDecision.Reason}, err
			}
			return Result{RequestID: cleanRequestID, Allowed: false, ApprovalRequired: true, Reason: approvalDecision.Reason}, nil
		}
	}
	if err != nil {
		return Result{RequestID: cleanRequestID, Allowed: false, ApprovalRequired: decision.ApprovalRequired, Reason: decision.Reason}, err
	}
	if len(requiredCapabilities) == 0 {
		return Result{RequestID: cleanRequestID, Allowed: true, ApprovalRequired: false, Reason: decision.Reason}, nil
	}
	if p == nil || p.Registry == nil {
		return Result{RequestID: cleanRequestID, Allowed: false, ApprovalRequired: false, Reason: "provider registry is not configured"}, fmt.Errorf("provider registry is not configured")
	}
	for _, id := range p.Registry.IDs() {
		if err := p.Registry.ValidateCapabilities(id, requiredCapabilities); err == nil {
			return Result{RequestID: cleanRequestID, ProviderID: id, Allowed: true, ApprovalRequired: false, Reason: decision.Reason}, nil
		}
	}
	return Result{RequestID: cleanRequestID, Allowed: false, ApprovalRequired: false, Reason: "no enabled provider satisfies the required capabilities"}, fmt.Errorf("no enabled provider satisfies the required capabilities")
}

// Generate validates the request and, if allowed, invokes the selected provider.
func (p *Processor) Generate(ctx context.Context, requestID, purpose, executionMode string, req model.Request, requiredCapabilities []string) (model.Response, error) {
	result, err := p.Process(requestID, purpose, executionMode, requiredCapabilities)
	if err != nil || !result.Allowed {
		if err == nil {
			err = fmt.Errorf("request %q is not allowed", requestID)
		}
		return model.Response{}, err
	}
	if p == nil || p.Registry == nil {
		return model.Response{}, fmt.Errorf("provider registry is not configured")
	}
	provider, ok := p.Registry.Lookup(result.ProviderID)
	if !ok {
		return model.Response{}, fmt.Errorf("provider %q not registered", result.ProviderID)
	}
	if err := p.Registry.ValidateCapabilities(result.ProviderID, requiredCapabilities); err != nil {
		return model.Response{}, err
	}
	return provider.Generate(ctx, req)
}
