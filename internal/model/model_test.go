package model

import (
	"context"
	"testing"
)

func TestProviderRegistryAcceptsEnabledProvider(t *testing.T) {
	provider := &MockProvider{Name: "mock-provider", Active: true}
	registry := NewRegistry()
	registry.Register(provider)
	if err := registry.Validate("mock-provider"); err != nil {
		t.Fatalf("provider validation error: %v", err)
	}
}

func TestProviderRegistryRejectsDisabledProvider(t *testing.T) {
	provider := &MockProvider{Name: "mock-provider", Active: false}
	registry := NewRegistry()
	registry.Register(provider)
	if err := registry.Validate("mock-provider"); err == nil {
		t.Fatal("disabled provider should be rejected")
	}
}

func TestModelRequestNormalization(t *testing.T) {
	req := Request{ModelID: "gpt-4o-mini", Prompt: "review repo", Metadata: map[string]string{"trace": "abc"}}
	if req.ModelID == "" || req.Prompt == "" {
		t.Fatal("normalized request must contain model and prompt")
	}
}

func TestProviderRegistryRejectsMissingCapability(t *testing.T) {
	provider := &MockProvider{Name: "mock-provider", Active: true}
	registry := NewRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatalf("register error: %v", err)
	}
	if err := registry.ValidateCapabilities("mock-provider", []string{"reasoning"}); err == nil {
		t.Fatal("provider without reasoning capability should be rejected")
	}
}

func TestProviderRegistryAllowsMatchingCapability(t *testing.T) {
	provider := &MockProvider{Name: "mock-provider", Active: true}
	registry := NewRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatalf("register error: %v", err)
	}
	if err := registry.ValidateCapabilities("mock-provider", []string{"text_generation"}); err != nil {
		t.Fatalf("required capability should be accepted: %v", err)
	}
}

type MockProvider struct {
	Name   string
	Active bool
}

func (p *MockProvider) ID() string    { return p.Name }
func (p *MockProvider) Enabled() bool { return p.Active }

func (p *MockProvider) Generate(ctx context.Context, req Request) (Response, error) {
	return Response{ModelID: req.ModelID, Text: "ok"}, nil
}

func (p *MockProvider) HealthCheck(ctx context.Context) error { return nil }

func (p *MockProvider) Capabilities() []string { return []string{"text_generation"} }
