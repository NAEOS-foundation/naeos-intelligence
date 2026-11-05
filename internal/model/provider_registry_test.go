package model

import (
	"context"
	"testing"
)

type registryTestProvider struct {
	id string
}

func (p *registryTestProvider) ID() string { return p.id }
func (p *registryTestProvider) Generate(context.Context, Request) (Response, error) {
	return Response{ModelID: "test", Text: p.id}, nil
}
func (p *registryTestProvider) HealthCheck(context.Context) error { return nil }
func (p *registryTestProvider) Capabilities() []string             { return nil }

func TestRegistryRejectsDuplicateProviderIDWithoutReplacingExisting(t *testing.T) {
	registry := NewRegistry()
	original := &registryTestProvider{id: "same-provider"}
	replacement := &registryTestProvider{id: "same-provider"}

	if err := registry.Register(original); err != nil {
		t.Fatalf("register original: %v", err)
	}
	if err := registry.Register(replacement); err == nil {
		t.Fatal("register duplicate ID should fail")
	}

	got, ok := registry.Lookup("same-provider")
	if !ok {
		t.Fatal("original provider should remain registered")
	}
	if got != original {
		t.Fatal("duplicate registration replaced the original provider")
	}
}
