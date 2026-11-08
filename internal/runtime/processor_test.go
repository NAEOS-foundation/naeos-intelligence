package runtime

import (
	"context"
	"testing"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/tool"
)

func TestProcessorAcceptsAnalysisOnlyRequestWithMatchingCapability(t *testing.T) {
	registry := model.NewRegistry()
	if err := registry.Register(&testProvider{id: "mock-provider", active: true, capabilities: []string{"text_generation"}}); err != nil {
		t.Fatalf("register error: %v", err)
	}
	processor := NewProcessor(registry, nil)
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
	processor := NewProcessor(registry, nil)
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
	processor := NewProcessor(registry, nil)
	processor.ApprovalWorkflow = workflow
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
	processor := NewProcessor(registry, nil)
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

func TestProcessorSelectsDeterministicProviderWithoutRequiredCapabilities(t *testing.T) {
	registry := model.NewRegistry()
	for _, id := range []string{"provider-z", "provider-a"} {
		if err := registry.Register(&testProvider{id: id, active: true}); err != nil {
			t.Fatalf("register %s: %v", id, err)
		}
	}
	processor := NewProcessor(registry, nil)
	result, err := processor.Process("req-201", "analyze context", "analysis_only", nil)
	if err != nil {
		t.Fatalf("process error: %v", err)
	}
	if !result.Allowed {
		t.Fatal("request with no required capabilities should be allowed when an enabled provider exists")
	}
	if result.ProviderID != "provider-a" {
		t.Fatalf("provider_id=%q, want deterministic first provider provider-a", result.ProviderID)
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

type testToolForRuntime struct {
	id      string
	enabled bool
	caps    []string
}

func (t *testToolForRuntime) ID() string   { return t.id }
func (t *testToolForRuntime) Name() string { return t.id }
func (t *testToolForRuntime) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	return map[string]any{"result": "ok"}, nil
}
func (t *testToolForRuntime) Validate(input map[string]any) error { return nil }
func (t *testToolForRuntime) Spec() tool.ToolSpec {
	return tool.ToolSpec{ID: t.id, Enabled: t.enabled, Capabilities: t.caps}
}
func (t *testToolForRuntime) Enabled() bool { return t.enabled }

func TestProcessorValidateToolRegisteredAndEnabled(t *testing.T) {
	toolRegistry := tool.NewRegistry()
	tk := &testToolForRuntime{id: "scan-tool", enabled: true, caps: []string{"read"}}
	if err := toolRegistry.Register(tk); err != nil {
		t.Fatalf("register error: %v", err)
	}
	processor := NewProcessor(model.NewRegistry(), toolRegistry)
	if err := processor.ValidateTool("scan-tool", []string{"read"}); err != nil {
		t.Fatalf("validate tool error: %v", err)
	}
}

func TestProcessorValidateToolNotRegistered(t *testing.T) {
	processor := NewProcessor(model.NewRegistry(), tool.NewRegistry())
	if err := processor.ValidateTool("nonexistent", nil); err == nil {
		t.Fatal("expected error for not registered tool")
	}
}

func TestProcessorValidateToolMissingCapability(t *testing.T) {
	toolRegistry := tool.NewRegistry()
	tk := &testToolForRuntime{id: "scan-tool", enabled: true, caps: []string{"read"}}
	if err := toolRegistry.Register(tk); err != nil {
		t.Fatalf("register error: %v", err)
	}
	processor := NewProcessor(model.NewRegistry(), toolRegistry)
	if err := processor.ValidateTool("scan-tool", []string{"delete"}); err == nil {
		t.Fatal("expected error for missing capability")
	}
}
