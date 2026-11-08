// Package tool provides the tool registry and capability contracts for
// governed execution. Tools are explicitly registered, enabled/disabled, and
// validated before any execution path may invoke them.
package tool

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// ToolSpec describes a registered tool's capabilities and constraints.
type ToolSpec struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Category         string   `json:"category"`
	Capabilities     []string `json:"capabilities"`
	Enabled          bool     `json:"enabled"`
	ReadOnly         bool     `json:"read_only"`
	RequiresApproval bool     `json:"requires_approval"`
}

// Tool is the narrow adapter contract for governed tool execution.
type Tool interface {
	ID() string
	Name() string
	Execute(ctx context.Context, input map[string]any) (map[string]any, error)
	Validate(input map[string]any) error
	Spec() ToolSpec
}

// EnabledTool is an optional capability interface for runtime enable/disable.
type EnabledTool interface {
	Enabled() bool
}

// Registry tracks tool instances that may be selected for governed execution.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry. Existing identifiers are rejected
// to prevent an accidental or unreviewed replacement of a registered tool.
func (r *Registry) Register(t Tool) error {
	if t == nil {
		return errors.New("tool is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tools == nil {
		r.tools = make(map[string]Tool)
	}
	id := toolID(t)
	if id == "" {
		return errors.New("tool has no identifier")
	}
	if _, exists := r.tools[id]; exists {
		return fmt.Errorf("tool %q already registered", id)
	}
	r.tools[id] = t
	return nil
}

// Validate ensures the tool exists and is enabled before selection.
func (r *Registry) Validate(id string) error {
	if id == "" {
		return errors.New("tool id is required")
	}
	r.mu.RLock()
	t, ok := r.tools[id]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("tool %q not registered", id)
	}
	if checker, ok := t.(EnabledTool); ok && !checker.Enabled() {
		return fmt.Errorf("tool %q is disabled", id)
	}
	return nil
}

// ValidateCapabilities ensures a tool is registered, enabled, and offers the required capability set.
func (r *Registry) ValidateCapabilities(id string, required []string) error {
	if err := r.Validate(id); err != nil {
		return err
	}
	if len(required) == 0 {
		return nil
	}
	t, ok := r.lookup(id)
	if !ok {
		return fmt.Errorf("tool %q not registered", id)
	}
	capabilities := make(map[string]struct{}, len(t.Spec().Capabilities))
	for _, cap := range t.Spec().Capabilities {
		capabilities[cap] = struct{}{}
	}
	for _, cap := range required {
		if _, ok := capabilities[cap]; !ok {
			return fmt.Errorf("tool %q does not support capability %q", id, cap)
		}
	}
	return nil
}

// IDs returns the registered tool identifiers in deterministic order.
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.tools))
	for id := range r.tools {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r *Registry) lookup(id string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[id]
	return t, ok
}

// Lookup returns the registered tool instance for the given identifier.
func (r *Registry) Lookup(id string) (Tool, bool) {
	return r.lookup(id)
}

// Enable marks a tool as enabled for execution.
func (r *Registry) Enable(id string) error {
	t, ok := r.lookup(id)
	if !ok {
		return fmt.Errorf("tool %q not registered", id)
	}
	setter, ok := t.(interface{ SetEnabled(bool) })
	if !ok {
		return fmt.Errorf("tool %q does not support enable/disable", id)
	}
	setter.SetEnabled(true)
	return nil
}

// Disable marks a tool as disabled for execution.
func (r *Registry) Disable(id string) error {
	t, ok := r.lookup(id)
	if !ok {
		return fmt.Errorf("tool %q not registered", id)
	}
	setter, ok := t.(interface{ SetEnabled(bool) })
	if !ok {
		return fmt.Errorf("tool %q does not support enable/disable", id)
	}
	setter.SetEnabled(false)
	return nil
}

func toolID(t Tool) string {
	return t.ID()
}
