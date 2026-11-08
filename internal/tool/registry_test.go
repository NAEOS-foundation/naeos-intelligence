package tool

import (
	"context"
	"testing"
)

type testTool struct {
	id               string
	name             string
	enabled          bool
	readOnly         bool
	requiresApproval bool
	caps             []string
}

func (t *testTool) ID() string   { return t.id }
func (t *testTool) Name() string { return t.name }
func (t *testTool) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	return map[string]any{"result": "ok"}, nil
}
func (t *testTool) Validate(input map[string]any) error { return nil }
func (t *testTool) Spec() ToolSpec {
	return ToolSpec{
		ID:               t.id,
		Name:             t.name,
		Enabled:          t.enabled,
		ReadOnly:         t.readOnly,
		RequiresApproval: t.requiresApproval,
		Capabilities:     t.caps,
	}
}
func (t *testTool) Enabled() bool     { return t.enabled }
func (t *testTool) SetEnabled(e bool) { t.enabled = e }

func TestRegistryRejectsDuplicateToolIDWithoutReplacingExisting(t *testing.T) {
	registry := NewRegistry()
	original := &testTool{id: "same-tool", name: "same", enabled: true}
	replacement := &testTool{id: "same-tool", name: "replaced", enabled: true}

	if err := registry.Register(original); err != nil {
		t.Fatalf("register original: %v", err)
	}
	if err := registry.Register(replacement); err == nil {
		t.Fatal("register duplicate ID should fail")
	}

	got, ok := registry.Lookup("same-tool")
	if !ok {
		t.Fatal("original tool should remain registered")
	}
	if got != original {
		t.Fatal("duplicate registration replaced the original tool")
	}
}

func TestRegistryValidateNotRegistered(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Validate("nonexistent"); err == nil {
		t.Fatal("expected error for not registered tool")
	}
}

func TestRegistryValidateDisabledTool(t *testing.T) {
	registry := NewRegistry()
	disabled := &testTool{id: "disabled-tool", name: "disabled", enabled: false}
	if err := registry.Register(disabled); err != nil {
		t.Fatalf("register disabled: %v", err)
	}
	if err := registry.Validate("disabled-tool"); err == nil {
		t.Fatal("expected error for disabled tool")
	}
}

func TestRegistryValidateCapabilities(t *testing.T) {
	registry := NewRegistry()
	capable := &testTool{id: "cap-tool", name: "capable", enabled: true, caps: []string{"read", "write"}}
	if err := registry.Register(capable); err != nil {
		t.Fatalf("register cap-tool: %v", err)
	}

	if err := registry.ValidateCapabilities("cap-tool", []string{"read"}); err != nil {
		t.Fatalf("expected no error for required capability read: %v", err)
	}
	if err := registry.ValidateCapabilities("cap-tool", []string{"read", "write"}); err != nil {
		t.Fatalf("expected no error for required capabilities read+write: %v", err)
	}
	if err := registry.ValidateCapabilities("cap-tool", []string{"delete"}); err == nil {
		t.Fatal("expected error for missing capability delete")
	}
}

func TestRegistryIDsAreSorted(t *testing.T) {
	registry := NewRegistry()
	for _, id := range []string{"zebra", "alpha", "midway"} {
		if err := registry.Register(&testTool{id: id, name: id, enabled: true}); err != nil {
			t.Fatalf("register %s: %v", id, err)
		}
	}
	ids := registry.IDs()
	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(ids))
	}
	if ids[0] != "alpha" || ids[1] != "midway" || ids[2] != "zebra" {
		t.Fatalf("ids not sorted: %v", ids)
	}
}

func TestRegistryEnableDisable(t *testing.T) {
	registry := NewRegistry()
	toggle := &testTool{id: "toggle-tool", name: "toggle", enabled: true}
	if err := registry.Register(toggle); err != nil {
		t.Fatalf("register toggle-tool: %v", err)
	}

	if err := registry.Disable("toggle-tool"); err != nil {
		t.Fatalf("disable toggle-tool: %v", err)
	}
	if err := registry.Validate("toggle-tool"); err == nil {
		t.Fatal("expected error after disable")
	}

	if err := registry.Enable("toggle-tool"); err != nil {
		t.Fatalf("enable toggle-tool: %v", err)
	}
	if err := registry.Validate("toggle-tool"); err != nil {
		t.Fatalf("expected no error after enable: %v", err)
	}
}

func TestRegistryLookupNotFound(t *testing.T) {
	registry := NewRegistry()
	if _, ok := registry.Lookup("nonexistent"); ok {
		t.Fatal("expected not found for nonexistent tool")
	}
}
