package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/evidence"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/classifier"
	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/planner"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/policy"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/runtime"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/verification"
)

// PipelineResult is the full orchestration output for a read-only intelligence request.
type PipelineResult struct {
	RequestID      string            `json:"request_id"`
	Intent         classifier.Intent `json:"intent"`
	Plan           planner.Plan      `json:"plan"`
	RuntimeResult  runtime.Result    `json:"runtime_result"`
	PolicyDecision policy.Decision   `json:"policy_decision"`
	Provenance     string            `json:"provenance"`
	CreatedAt      time.Time         `json:"created_at"`
}

// Orchestrator coordinates classifier, context builder, planner, and runtime.
type Orchestrator struct {
	Classifier         classifier.Classifier
	Context            intctx.Builder
	Planner            planner.Planner
	Processor          *runtime.Processor
	Approval           *governance.ApprovalWorkflow
	Audit              *governance.AuditLogger
	EvidenceStore      *evidence.Store
	VerificationEngine *verification.Engine
}

// NewOrchestrator creates an orchestrator wiring intelligence components together.
func NewOrchestrator(cl classifier.Classifier, cb intctx.Builder, pl planner.Planner, rp *runtime.Processor) *Orchestrator {
	return &Orchestrator{
		Classifier: cl,
		Context:    cb,
		Planner:    pl,
		Processor:  rp,
	}
}

// NewOrchestratorWithEvidence creates an orchestrator with evidence and verification wired in.
func NewOrchestratorWithEvidence(cl classifier.Classifier, cb intctx.Builder, pl planner.Planner, rp *runtime.Processor, es *evidence.Store, ve *verification.Engine) *Orchestrator {
	return &Orchestrator{
		Classifier:         cl,
		Context:            cb,
		Planner:            pl,
		Processor:          rp,
		EvidenceStore:      es,
		VerificationEngine: ve,
	}
}

// Run executes the read-only intelligence pipeline: classify → context → plan → runtime gate.
func (o *Orchestrator) Run(ctx context.Context, clientID, requestID, purpose, executionMode string, inputs []intctx.SourceInput, desiredOutputs []string, req model.Request, requiredCapabilities []string) (PipelineResult, error) {
	if err := ctx.Err(); err != nil {
		return PipelineResult{}, err
	}
	if strings.TrimSpace(clientID) == "" {
		return PipelineResult{}, errors.New("client ID is required")
	}
	if strings.TrimSpace(requestID) == "" {
		return PipelineResult{}, errors.New("request_id is required")
	}
	if strings.TrimSpace(purpose) == "" {
		return PipelineResult{}, errors.New("purpose is required")
	}
	if o.Classifier == nil {
		return PipelineResult{}, errors.New("classifier is required")
	}
	if o.Context == nil {
		return PipelineResult{}, errors.New("context builder is required")
	}
	if o.Planner == nil {
		return PipelineResult{}, errors.New("planner is required")
	}

	now := time.Now().UTC()

	// 1. Classify intent
	classSources := make([]classifier.ContextSource, 0, len(inputs))
	for _, in := range inputs {
		classSources = append(classSources, classifier.ContextSource{Type: in.Type, Reference: in.Reference})
	}
	classResult, err := o.Classifier.Classify(ctx, purpose, classSources)
	if err != nil {
		return PipelineResult{}, fmt.Errorf("classify: %w", err)
	}

	// 2. Build context
	builtCtx, err := o.Context.Build(ctx, requestID, inputs)
	if err != nil {
		return PipelineResult{}, fmt.Errorf("build context: %w", err)
	}

	// 3. Plan
	plan, err := o.Planner.Plan(ctx, requestID, classResult.Intent, &builtCtx, desiredOutputs)
	if err != nil {
		return PipelineResult{}, fmt.Errorf("plan: %w", err)
	}

	// 4. Runtime gate (read-only policy + provider selection)
	var rtResult runtime.Result
	if o.Processor != nil {
		rtResult, err = o.Processor.Process(requestID, purpose, executionMode, requiredCapabilities)
		if err != nil {
			rtResult = runtime.Result{RequestID: requestID, Allowed: false, Reason: err.Error()}
		}
	} else {
		decision, polErr := policy.EvaluateRequest(policy.RequestInput{Purpose: purpose, ExecutionMode: executionMode})
		rtResult = runtime.Result{RequestID: requestID, Allowed: decision.Allowed, ApprovalRequired: decision.ApprovalRequired, Reason: decision.Reason}
		if polErr != nil {
			return PipelineResult{}, fmt.Errorf("policy: %w", polErr)
		}
	}

	// 5. Record audit
	if o.Audit != nil {
		_ = o.Audit.Record(requestID, "orchestration.completed", fmt.Sprintf("intent=%s allowed=%v", classResult.Intent, rtResult.Allowed))
	}

	// 6. Record evidence events (fail-safe; does not fail the pipeline)
	if o.EvidenceStore != nil {
		_ = o.EvidenceStore.RecordEvent(ctx, clientID, requestID, "orchestration.classified", "system", "orchestrator", fmt.Sprintf("intent=%s", classResult.Intent), "")
		_ = o.EvidenceStore.RecordEvent(ctx, clientID, requestID, "orchestration.planned", "system", "orchestrator", fmt.Sprintf("tasks=%d", len(plan.Tasks)), "")
		_ = o.EvidenceStore.RecordEvent(ctx, clientID, requestID, "orchestration.completed", "system", "orchestrator", fmt.Sprintf("intent=%s allowed=%v", classResult.Intent, rtResult.Allowed), "")
	}

	// 7. Run verification and record result as evidence (fail-safe)
	if o.VerificationEngine != nil && o.EvidenceStore != nil {
		vr, verr := o.VerificationEngine.Verify(ctx, requestID, string(classResult.Intent), string(plan.Intent))
		if verr == nil {
			_ = o.EvidenceStore.RecordEvent(ctx, clientID, requestID, "orchestration.verified", "system", "verification", fmt.Sprintf("result=%s expected=%s observed=%s", vr.Result, vr.ExpectedOutcome, vr.ObservedOutcome), vr.VerificationID)
		}
	}

	return PipelineResult{
		RequestID:      requestID,
		Intent:         classResult.Intent,
		Plan:           plan,
		RuntimeResult:  rtResult,
		PolicyDecision: policy.Decision{Allowed: rtResult.Allowed, ApprovalRequired: rtResult.ApprovalRequired, Reason: rtResult.Reason},
		Provenance:     "orchestrator pipeline; classifier=rule-based; context=rule-based; planner=rule-based",
		CreatedAt:      now,
	}, nil
}
