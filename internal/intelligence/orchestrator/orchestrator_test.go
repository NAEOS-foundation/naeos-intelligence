package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/classifier"
	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/planner"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/runtime"
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
	return planner.Plan{RequestID: requestID, Intent: intent, Tasks: []planner.Task{{ID: "t1", Title: "task", Status: "proposed"}}}, nil
}

func TestOrchestrator_Run(t *testing.T) {
	cl := &fakeClassifier{intent: classifier.IntentRepositoryAnalysis}
	cb := &fakeContextBuilder{}
	pl := &fakePlanner{}
	o := NewOrchestrator(cl, cb, pl, nil)

	res, err := o.Run(context.Background(), "req-1", "analyze repository architecture", "analysis_only",
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
	o := NewOrchestrator(cl, cb, pl, nil)

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
			_, err := o.Run(context.Background(), tt.requestID, tt.purpose, "analysis_only", nil, nil, model.Request{}, nil)
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
	o := NewOrchestrator(&fakeClassifier{err: errors.New("classify failed")}, &fakeContextBuilder{}, &fakePlanner{}, nil)
	_, err := o.Run(context.Background(), "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil || err.Error() != "classify: classify failed" {
		t.Errorf("unexpected error: %v", err)
	}

	o = NewOrchestrator(&fakeClassifier{intent: classifier.IntentRepositoryAnalysis}, &fakeContextBuilder{err: errors.New("build failed")}, &fakePlanner{}, nil)
	_, err = o.Run(context.Background(), "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil || err.Error() != "build context: build failed" {
		t.Errorf("unexpected error: %v", err)
	}

	o = NewOrchestrator(&fakeClassifier{intent: classifier.IntentRepositoryAnalysis}, &fakeContextBuilder{}, &fakePlanner{err: errors.New("plan failed")}, nil)
	_, err = o.Run(context.Background(), "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil || err.Error() != "plan: plan failed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOrchestrator_Run_ContextCancelled(t *testing.T) {
	o := NewOrchestrator(&fakeClassifier{intent: classifier.IntentRepositoryAnalysis}, &fakeContextBuilder{}, &fakePlanner{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := o.Run(ctx, "req-1", "purpose", "analysis_only", nil, nil, model.Request{}, nil)
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}

func TestOrchestrator_Run_WithProcessor(t *testing.T) {
	reg := model.NewRegistry()
	_ = reg.Register(&mockProvider{id: "test-provider"})
	proc := runtime.NewProcessor(reg)
	cl := &fakeClassifier{intent: classifier.IntentRepositoryAnalysis}
	cb := &fakeContextBuilder{}
	pl := &fakePlanner{}
	o := NewOrchestrator(cl, cb, pl, proc)
	o.Approval = governance.NewApprovalWorkflow()
	o.Audit = governance.NewAuditLogger()

	res, err := o.Run(context.Background(), "req-1", "analyze repository architecture", "analysis_only", nil, nil, model.Request{}, nil)
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
