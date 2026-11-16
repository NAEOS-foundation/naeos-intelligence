package orchestrator

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/evidence"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/classifier"
	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/planner"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/runtime"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/verification"
)

// fakeClassifier returns a fixed intent.
type fakeClassifier struct {
	intent classifier.Intent
	err    error
}

func (f *fakeClassifier) Classify(ctx context.Context, purpose string, sources []classifier.ContextSource) (classifier.Result, error) {
	if f.err != nil {
		return classifier.Result{}, f.err
	}
	return classifier.Result{Intent: f.intent, Confidence: 0.8, Reason: "fake", Evidence: nil}, nil
}

// fakeContextBuilder returns a fixed context.
type fakeContextBuilder struct {
	err error
}

func (f *fakeContextBuilder) Build(ctx context.Context, requestID string, inputs []intctx.SourceInput) (intctx.Context, error) {
	if f.err != nil {
		return intctx.Context{}, f.err
	}
	return intctx.Context{RequestID: requestID, Provenance: "fake"}, nil
}

// fakePlanner returns a fixed plan.
type fakePlanner struct {
	plan planner.Plan
	err  error
}

func (f *fakePlanner) Plan(ctx context.Context, requestID string, intent classifier.Intent, cx *intctx.Context, outputs []string) (planner.Plan, error) {
	if f.err != nil {
		return planner.Plan{}, f.err
	}
	if len(f.plan.Tasks) > 0 {
		f.plan.RequestID = requestID
		return f.plan, nil
	}
	return planner.Plan{RequestID: requestID, Intent: intent, Tasks: []planner.Task{{ID: "t1", Title: "task", Status: "proposed"}}}, nil
}

func newTestOrchestrator(cl classifier.Classifier, cb intctx.Builder, pl planner.Planner, rp *runtime.Processor) *Orchestrator {
	o := NewOrchestrator(cl, cb, pl, rp)
	o.EvidenceStore = &fakeEvidenceRecorder{}
	return o
}

func TestOrchestrator_Run(t *testing.T) {
	cl := &fakeClassifier{intent: classifier.IntentRepositoryAnalysis}
	cb := &fakeContextBuilder{}
	pl := &fakePlanner{}
	o := newTestOrchestrator(cl, cb, pl, nil)

	res, err := o.Run(context.Background(), "client-a", "req-1", "analyze repository architecture", "analysis_only",
		[]intctx.SourceInput{{Type: "repository", Reference: "github.com/o/r"}},
		nil, model.Request{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RequestID != "req-1" {
		t.Errorf("request_id = %q, want req-1", res.RequestID)
	}
	if res.Intent != classifier.IntentRepositoryAnalysis {
		t.Errorf("intent = %q, want repository_analysis", res.Intent)
	}
	if len(res.Plan.Tasks) == 0 {
		t.Error("plan must contain at least one task")
	}
	if !res.RuntimeResult.Allowed {
		t.Error("runtime result must be allowed for analysis_only")
	}
	if res.Provenance == "" {
		t.Error("provenance must not be empty")
	}
	if res.CreatedAt.IsZero() {
		t.Error("created_at must not be zero")
	}
}

func TestOrchestrator_Run_ValidationErrors(t *testing.T) {
	cl := &fakeClassifier{intent: classifier.IntentRepositoryAnalysis}
	cb := &fakeContextBuilder{}
	pl := &fakePlanner{}
	o := newTestOrchestrator(cl, cb, pl, nil)

	tests := []struct {
		name      string
		requestID string
		purpose   string
		wantErr   bool
	}{
		{"empty request id", "", "purpose", true},
		{"empty purpose", "req-1", "", true},
		{"valid", "req-1", "purpose", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := o.Run(context.Background(), "client-a", tt.requestID, tt.purpose, "analysis_only", nil, nil, model.Request{}, nil)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestOrchestrator_Run_ComponentErrors(t *testing.T) {
	o := newTestOrchestrator(&fakeClassifier{err: errors.New("classify failed")}, &fakeContextBuilder{}, &fakePlanner{}, nil)
	_, err := o.Run(context.Background(), "client-a", "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil || err.Error() != "classify: classify failed" {
		t.Errorf("unexpected error: %v", err)
	}

	o = newTestOrchestrator(&fakeClassifier{intent: classifier.IntentRepositoryAnalysis}, &fakeContextBuilder{err: errors.New("build failed")}, &fakePlanner{}, nil)
	_, err = o.Run(context.Background(), "client-a", "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil || err.Error() != "build context: build failed" {
		t.Errorf("unexpected error: %v", err)
	}

	o = newTestOrchestrator(&fakeClassifier{intent: classifier.IntentRepositoryAnalysis}, &fakeContextBuilder{}, &fakePlanner{err: errors.New("plan failed")}, nil)
	_, err = o.Run(context.Background(), "client-a", "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil || err.Error() != "plan: plan failed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOrchestrator_Run_ContextCancelled(t *testing.T) {
	o := newTestOrchestrator(&fakeClassifier{intent: classifier.IntentRepositoryAnalysis}, &fakeContextBuilder{}, &fakePlanner{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := o.Run(ctx, "client-a", "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}

func TestOrchestrator_Run_WithProcessor(t *testing.T) {
	reg := model.NewRegistry()
	_ = reg.Register(&mockProvider{id: "test-provider"})
	proc := runtime.NewProcessor(reg, nil)
	cl := &fakeClassifier{intent: classifier.IntentRepositoryAnalysis}
	cb := &fakeContextBuilder{}
	pl := &fakePlanner{}
	o := newTestOrchestrator(cl, cb, pl, proc)
	o.Approval = governance.NewApprovalWorkflow()
	o.Audit = governance.NewAuditLogger()

	res, err := o.Run(context.Background(), "client-a", "req-1", "analyze repository architecture", "analysis_only", nil, nil, model.Request{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.RuntimeResult.Allowed {
		t.Error("runtime result must be allowed for analysis_only with provider")
	}
	if res.PolicyDecision.Allowed != true {
		t.Error("policy decision must be allowed")
	}
	if res.Provenance == "" {
		t.Error("provenance must not be empty")
	}
}

func TestOrchestrator_Run_AllIntentsProposed(t *testing.T) {
	reg := model.NewRegistry()
	_ = reg.Register(&mockProvider{id: "eval-provider"})
	proc := runtime.NewProcessor(reg, nil)
	cl := classifier.NewRuleClassifier()
	cb := intctx.NewRuleBuilder()
	pl := planner.NewRulePlanner()
	o := newTestOrchestrator(cl, cb, pl, proc)

	intents := []classifier.Intent{
		classifier.IntentRepositoryAnalysis,
		classifier.IntentIssueInterpretation,
		classifier.IntentPRReview,
		classifier.IntentPlanning,
		classifier.IntentKnowledgeRetrieval,
		classifier.IntentUnknown,
	}
	purposes := []string{
		"analyze repository architecture and technical debt",
		"interpret this engineering issue and requirement",
		"review the pull request diff and identify risks",
		"generate a structured execution plan with dependencies",
		"retrieve documentation and context from knowledge base",
		"something completely unrelated xyzzy",
	}
	sourceSets := [][]intctx.SourceInput{
		{{Type: "repository", Reference: "github.com/o/r"}},
		{{Type: "repository", Reference: "github.com/o/r"}},
		{{Type: "repository", Reference: "github.com/o/r"}},
		{{Type: "repository", Reference: "github.com/o/r"}},
		{{Type: "repository", Reference: "github.com/o/r"}},
		nil,
	}
	for i, intent := range intents {
		res, err := o.Run(context.Background(), "client-a", "eval-req-"+string(rune('1'+i)), purposes[i], "analysis_only",
			sourceSets[i], nil, model.Request{}, nil)
		if err != nil {
			t.Fatalf("intent %s: unexpected error: %v", intent, err)
		}
		if res.Intent != intent {
			t.Errorf("intent %s: got %q, want %q", intent, res.Intent, intent)
		}
		if len(res.Plan.Tasks) == 0 {
			t.Errorf("intent %s: plan must contain at least one task", intent)
		}
		for _, task := range res.Plan.Tasks {
			if task.Status != "proposed" {
				t.Errorf("intent %s: task %s status = %q, want proposed", intent, task.ID, task.Status)
			}
		}
		if !res.RuntimeResult.Allowed {
			t.Errorf("intent %s: runtime must be allowed for analysis_only", intent)
		}
		if res.PolicyDecision.Allowed != true {
			t.Errorf("intent %s: policy must be allowed", intent)
		}
	}
}

func TestOrchestrator_Run_Deterministic(t *testing.T) {
	reg := model.NewRegistry()
	_ = reg.Register(&mockProvider{id: "det-provider"})
	proc := runtime.NewProcessor(reg, nil)
	cl := classifier.NewRuleClassifier()
	cb := intctx.NewRuleBuilder()
	pl := planner.NewRulePlanner()
	o := newTestOrchestrator(cl, cb, pl, proc)

	purpose := "analyze repository architecture and technical debt"
	inputs := []intctx.SourceInput{{Type: "repository", Reference: "github.com/o/r"}}

	first, err := o.Run(context.Background(), "client-a", "det-req-1", purpose, "analysis_only", inputs, nil, model.Request{}, nil)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := o.Run(context.Background(), "client-a", "det-req-2", purpose, "analysis_only", inputs, nil, model.Request{}, nil)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if first.Intent != second.Intent {
		t.Errorf("intent not deterministic: %q vs %q", first.Intent, second.Intent)
	}
	if len(first.Plan.Tasks) != len(second.Plan.Tasks) {
		t.Errorf("task count not deterministic: %d vs %d", len(first.Plan.Tasks), len(second.Plan.Tasks))
	}
	if first.Provenance != second.Provenance {
		t.Errorf("provenance not deterministic: %q vs %q", first.Provenance, second.Provenance)
	}
}

type mockProvider struct {
	id string
}

func (m *mockProvider) ID() string { return m.id }
func (m *mockProvider) Generate(ctx context.Context, req model.Request) (model.Response, error) {
	return model.Response{ModelID: m.id, Text: "fake"}, nil
}
func (m *mockProvider) HealthCheck(ctx context.Context) error { return nil }
func (m *mockProvider) Capabilities() []string                { return []string{"text"} }

func TestNewOrchestrator_NilFields(t *testing.T) {
	o := NewOrchestrator(nil, nil, nil, nil)
	if o.Classifier != nil || o.Context != nil || o.Planner != nil || o.Processor != nil {
		t.Error("fields should be assigned as-is even when nil")
	}
}

type fakeEvidenceRecorder struct {
	failEvent string
	err       error
	events    []string
	records   []evidence.Event
}

func (f *fakeEvidenceRecorder) RecordEvents(_ context.Context, _, _ string, events []evidence.Event) error {
	for _, event := range events {
		f.events = append(f.events, event.EventType)
		f.records = append(f.records, event)
		if event.EventType == f.failEvent {
			if f.err != nil {
				return f.err
			}
			return errors.New("evidence write failed")
		}
	}
	return nil
}

func TestOrchestrator_Run_FailsWhenEvidencePersistenceFails(t *testing.T) {
	writeErr := errors.New("database unavailable")
	evidence := &fakeEvidenceRecorder{failEvent: "orchestration.planned", err: writeErr}
	o := NewOrchestratorWithEvidence(
		&fakeClassifier{intent: classifier.IntentRepositoryAnalysis},
		&fakeContextBuilder{},
		&fakePlanner{},
		nil,
		evidence,
		nil,
	)
	o.Audit = governance.NewAuditLogger()

	res, err := o.Run(context.Background(), "client-a", "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil {
		t.Fatal("expected evidence persistence error")
	}
	if !errors.Is(err, writeErr) {
		t.Fatalf("error = %v, want wrapped database error", err)
	}
	if res.RequestID != "" {
		t.Fatalf("result = %+v, want empty result on persistence failure", res)
	}
	if got := len(o.Audit.Events()); got != 0 {
		t.Fatalf("audit events = %d, want no completion marker after failed evidence batch", got)
	}
	want := []string{"orchestration.classified", "orchestration.planned"}
	if len(evidence.events) != len(want) {
		t.Fatalf("events = %v, want %v", evidence.events, want)
	}
	for i := range want {
		if evidence.events[i] != want[i] {
			t.Fatalf("events = %v, want %v", evidence.events, want)
		}
	}
}

func TestOrchestrator_Run_RecordsCompletionLast(t *testing.T) {
	evidence := &fakeEvidenceRecorder{}
	o := NewOrchestratorWithEvidence(
		&fakeClassifier{intent: classifier.IntentRepositoryAnalysis},
		&fakeContextBuilder{},
		&fakePlanner{},
		nil,
		evidence,
		nil,
	)
	o.Audit = governance.NewAuditLogger()

	if _, err := o.Run(context.Background(), "client-a", "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := []string{"orchestration.classified", "orchestration.planned", "orchestration.completed"}
	if len(evidence.events) != len(want) {
		t.Fatalf("events = %v, want %v", evidence.events, want)
	}
	for i := range want {
		if evidence.events[i] != want[i] {
			t.Fatalf("events = %v, want %v", evidence.events, want)
		}
	}
	auditEvents := o.Audit.Events()
	if len(auditEvents) != 1 || auditEvents[0].EventType != "orchestration.completed" {
		t.Fatalf("audit events = %+v, want one completion marker after successful batch", auditEvents)
	}
}

func TestOrchestrator_Run_PersistsVerificationCorrelationID(t *testing.T) {
	recorder := &fakeEvidenceRecorder{}
	o := NewOrchestratorWithEvidence(
		&fakeClassifier{intent: classifier.IntentRepositoryAnalysis},
		&fakeContextBuilder{},
		&fakePlanner{},
		nil,
		recorder,
		verification.New(),
	)
	if _, err := o.Run(context.Background(), "client-a", "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, event := range recorder.records {
		if event.EventType == "orchestration.verified" {
			if !evidenceIDLooksLikeUUID(event.CorrelationID) {
				t.Fatalf("verification correlation ID = %q, want UUID", event.CorrelationID)
			}
			return
		}
	}
	t.Fatal("orchestration.verified evidence event was not persisted")
}

func evidenceIDLooksLikeUUID(value string) bool {
	return len(value) == 36 && value[8] == '-' && value[13] == '-' && value[18] == '-' && value[23] == '-'
}

func TestOrchestrator_Run_FailsClosedWhenVerificationFails(t *testing.T) {
	recorder := &fakeEvidenceRecorder{}
	failedPlan := planner.Plan{
		Intent: classifier.IntentUnknown,
		Tasks:  []planner.Task{{ID: "t1", Title: "task", Status: "proposed"}},
	}
	o := NewOrchestratorWithEvidence(
		&fakeClassifier{intent: classifier.IntentRepositoryAnalysis},
		&fakeContextBuilder{},
		&fakePlanner{plan: failedPlan},
		nil,
		recorder,
		verification.New(),
	)
	o.Audit = governance.NewAuditLogger()

	result, err := o.Run(context.Background(), "client-a", "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil {
		t.Fatal("Run() succeeded despite failed verification")
	}
	if result.RequestID != "" {
		t.Fatalf("Run() result = %+v, want empty result", result)
	}
	var sawFailedVerification, sawCompletion bool
	for _, event := range recorder.records {
		if event.EventType == "orchestration.verified" && strings.Contains(event.Message, "result=failed") {
			sawFailedVerification = true
			if event.CorrelationID == "" {
				t.Error("failed verification evidence has no correlation ID")
			}
		}
		if event.EventType == "orchestration.completed" {
			sawCompletion = true
		}
	}
	if !sawFailedVerification {
		t.Fatal("failed verification was not durably recorded")
	}
	if sawCompletion {
		t.Fatal("failed verification was incorrectly recorded as completed")
	}
	if len(o.Audit.Events()) != 0 {
		t.Fatal("in-memory completion marker was recorded after failed verification")
	}
}

func TestOrchestrator_Run_FailsClosedWithoutEvidenceRecorder(t *testing.T) {
	o := NewOrchestrator(
		&fakeClassifier{intent: classifier.IntentRepositoryAnalysis},
		&fakeContextBuilder{},
		&fakePlanner{},
		nil,
	)
	result, err := o.Run(context.Background(), "client-a", "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil || err.Error() != "durable evidence recorder is required" {
		t.Fatalf("Run() error = %v, want missing durable evidence recorder", err)
	}
	if result.RequestID != "" {
		t.Fatalf("Run() result = %+v, want empty result", result)
	}
}
