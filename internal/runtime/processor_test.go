package runtime

import (
	"context"
	"testing"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
)

func TestProcessorAcceptsAnalysisOnlyRequestWithMatchingCapability(t *testing.T) {
	registry := model.NewRegistry()
	if err := registry.Register(&testProvider{id: "mock-provider", active: true, capabilities: []string{"text_generation"}}); err != nil {
		t.Fatalf("register error: %v", err)
	}
	processor := NewProcessor(registry)
	result, err := processor.Process("req-100", "review repo", "analysis_only", []string{"text_generation"})
	if err != nil {
		t.Fatalf("process error: %v", err)
	}
	if !result.Allowed {
		t.Fatal("analysis_only request should be allowed")
	}
	if result.ProviderID != "mock-provider" {
		t.Fatalf("provider_id=%q, want mock-provider", result.ProviderID)
	}
}

func TestProcessorRejectsGovernedRequestBeforeProviderSelection(t *testing.T) {
	registry := model.NewRegistry()
	if err := registry.Register(&testProvider{id: "mock-provider", active: true, capabilities: []string{"text_generation"}}); err != nil {
		t.Fatalf("register error: %v", err)
	}
	processor := NewProcessor(registry)
	if _, err := processor.Process("req-101", "perform mutation", "governed", []string{"text_generation"}); err == nil {
		t.Fatal("governed request should be rejected before provider selection")
	}
}

func TestProcessorFlagsApprovalRequiredFromGovernanceWorkflow(t *testing.T) {
	registry := model.NewRegistry()
	if err := registry.Register(&testProvider{id: "mock-provider", active: true, capabilities: []string{"text_generation"}}); err != nil {
		t.Fatalf("register error: %v", err)
	}
	workflow := governance.NewApprovalWorkflow()
	processor := &Processor{Registry: registry, ApprovalWorkflow: workflow}
	result, err := processor.Process("req-approval", "perform mutation", "governed", []string{"text_generation"})
	if err == nil {
		t.Fatal("governed request should still be rejected in the MVP")
	}
	if !result.ApprovalRequired {
		t.Fatal("result should mark approval_required when the governance workflow flags governed execution")
	}
	if result.Reason == "" {
		t.Fatal("result should include the workflow reason")
	}
}

func TestProcessorGenerateUsesSelectedProvider(t *testing.T) {
	registry := model.NewRegistry()
	provider := &testProvider{id: "mock-provider", active: true, capabilities: []string{"text_generation"}}
	if err := registry.Register(provider); err != nil {
		t.Fatalf("register error: %v", err)
	}
	processor := NewProcessor(registry)
	resp, err := processor.Generate(context.Background(), "req-200", "review repo", "analysis_only", model.Request{ModelID: "demo-model", Prompt: "review repo"}, []string{"text_generation"})
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	if resp.ModelID != "demo-model" {
		t.Fatalf("model_id=%q, want demo-model", resp.ModelID)
	}
	if resp.Text != "ok" {
		t.Fatalf("text=%q, want ok", resp.Text)
	}
}

type testProvider struct {
	id           string
	active       bool
	capabilities []string
}

func (p *testProvider) ID() string    { return p.id }
func (p *testProvider) Enabled() bool { return p.active }
func (p *testProvider) Generate(ctx context.Context, req model.Request) (model.Response, error) {
	return model.Response{ModelID: req.ModelID, Text: "ok"}, nil
}
func (p *testProvider) HealthCheck(ctx context.Context) error { return nil }
func (p *testProvider) Capabilities() []string                { return p.capabilities }
