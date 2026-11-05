package model

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Request is the normalized model invocation contract used by the runtime.
type Request struct {
	ModelID  string
	Prompt   string
	Metadata map[string]string
}

// Response is the normalized output from a provider.
type Response struct {
	ModelID string
	Text    string
}

// Provider is the narrow adapter contract for model access.
type Provider interface {
	Generate(context.Context, Request) (Response, error)
	HealthCheck(context.Context) error
	Capabilities() []string
}

// EnabledProvider is an optional capability interface used by the registry.
type EnabledProvider interface {
	Enabled() bool
}

// Registry tracks provider instances that may be selected for use.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds a provider to the registry.
func (r *Registry) Register(provider Provider) error {
	if provider == nil {
		return errors.New("provider is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.providers == nil {
		r.providers = make(map[string]Provider)
	}
	id := providerID(provider)
	if id == "" {
		return errors.New("provider has no identifier")
	}
	r.providers[id] = provider
	return nil
}

// Validate ensures the provider exists and is enabled before selection.
func (r *Registry) Validate(id string) error {
	if id == "" {
		return errors.New("provider id is required")
	}
	r.mu.RLock()
	provider, ok := r.providers[id]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("provider %q not registered", id)
	}
	if checker, ok := provider.(EnabledProvider); ok && !checker.Enabled() {
		return fmt.Errorf("provider %q is disabled", id)
	}
	return nil
}

// ValidateCapabilities ensures a provider is registered, enabled, and offers the required capability set.
func (r *Registry) ValidateCapabilities(id string, required []string) error {
	if err := r.Validate(id); err != nil {
		return err
	}
	if len(required) == 0 {
		return nil
	}
	provider, ok := r.lookup(id)
	if !ok {
		return fmt.Errorf("provider %q not registered", id)
	}
	capabilities := make(map[string]struct{}, len(provider.Capabilities()))
	for _, cap := range provider.Capabilities() {
		capabilities[cap] = struct{}{}
	}
	for _, cap := range required {
		if _, ok := capabilities[cap]; !ok {
			return fmt.Errorf("provider %q does not support capability %q", id, cap)
		}
	}
	return nil
}

// IDs returns the registered provider identifiers in deterministic order.
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r *Registry) lookup(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[id]
	return provider, ok
}

// Lookup returns the registered provider instance for the given identifier.
func (r *Registry) Lookup(id string) (Provider, bool) {
	return r.lookup(id)
}

func providerID(provider Provider) string {
	switch p := provider.(type) {
	case interface{ ID() string }:
		return p.ID()
	case interface{ GetID() string }:
		return p.GetID()
	default:
		return ""
	}
}
