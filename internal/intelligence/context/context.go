package context

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/util"
)

// SourceType is a validated engineering context source type.
type SourceType string

const (
	SourceTypeRepository    SourceType = "repository"
	SourceTypeIssue         SourceType = "issue"
	SourceTypePullRequest   SourceType = "pull_request"
	SourceTypeDocumentation SourceType = "documentation"
	SourceTypeCommit        SourceType = "commit"
	SourceTypeNAEOSCore     SourceType = "naeos-core"
)

// Source is a validated, provenance-tracked context source.
type Source struct {
	Type        SourceType
	Reference   string
	Fingerprint string
	Metadata    map[string]any
	AddedAt     time.Time
}

// Context is the assembled engineering context for a request.
type Context struct {
	RequestID  string
	Sources    []Source
	Provenance string
	SizeBytes  int
	BuiltAt    time.Time
}

// Builder assembles context from declared sources with provenance.
type Builder interface {
	Build(ctx context.Context, requestID string, sources []SourceInput) (Context, error)
}

// SourceInput is an unvalidated source declaration from a request.
type SourceInput struct {
	Type      string
	Reference string
}

// ResolvedSource is a context source enriched with provider metadata.
type ResolvedSource struct {
	Type      string
	Reference string
	Metadata  map[string]any
}

// SourceResolver resolves a context source input into enriched metadata.
type SourceResolver interface {
	Resolve(ctx context.Context, input SourceInput) (ResolvedSource, error)
}

// SourceResolverSet routes each source type to its explicitly registered
// provider. Unregistered source types remain metadata-only and cause no
// outbound request.
type SourceResolverSet map[string]SourceResolver

// Resolve dispatches a source only to the provider registered for its type.
func (s SourceResolverSet) Resolve(ctx context.Context, input SourceInput) (ResolvedSource, error) {
	if resolver, ok := s[strings.TrimSpace(input.Type)]; ok && resolver != nil {
		return resolver.Resolve(ctx, input)
	}
	return ResolvedSource{Type: input.Type, Reference: input.Reference, Metadata: map[string]any{}}, nil
}

// ResolverOption configures a RuleBuilder.
type ResolverOption func(*RuleBuilder)

// WithResolver sets an optional source resolver that enriches context sources
// with provider metadata (e.g. GitHub repository data or NAEOS Core data)
// before fingerprinting.
func WithResolver(resolver SourceResolver) ResolverOption {
	return func(b *RuleBuilder) {
		b.resolver = resolver
	}
}

// RuleBuilder builds context using deterministic validation and fingerprinting.
type RuleBuilder struct {
	resolver SourceResolver
}

// NewRuleBuilder creates a deterministic context builder with optional resolvers.
func NewRuleBuilder(opts ...ResolverOption) *RuleBuilder {
	b := &RuleBuilder{}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

var validTypes = map[SourceType]struct{}{
	SourceTypeRepository:    {},
	SourceTypeIssue:         {},
	SourceTypePullRequest:   {},
	SourceTypeDocumentation: {},
	SourceTypeCommit:        {},
	SourceTypeNAEOSCore:     {},
}

func (b *RuleBuilder) Build(ctx context.Context, requestID string, inputs []SourceInput) (Context, error) {
	if err := ctx.Err(); err != nil {
		return Context{}, err
	}
	if strings.TrimSpace(requestID) == "" {
		return Context{}, errors.New("request_id is required")
	}
	if len(inputs) > 100 {
		return Context{}, errors.New("context sources must contain at most 100 items")
	}

	sources := make([]Source, 0, len(inputs))
	sizeBytes := 0
	now := time.Now().UTC()

	for _, in := range inputs {
		t := SourceType(strings.TrimSpace(in.Type))
		ref := strings.TrimSpace(in.Reference)
		if t == "" {
			return Context{}, errors.New("context source type is required")
		}
		if _, ok := validTypes[t]; !ok {
			return Context{}, errors.New("unsupported context source type: " + string(t))
		}
		if ref == "" || len(ref) > 1024 {
			return Context{}, errors.New("context source reference must contain 1 to 1024 characters")
		}

		metadata := map[string]any{}
		if b.resolver != nil {
			resolved, err := b.resolver.Resolve(ctx, in)
			if err != nil {
				return Context{}, fmt.Errorf("resolve source %s:%s: %w", in.Type, in.Reference, err)
			}
			metadata = resolved.Metadata
		}

		fp := fingerprint(string(t), ref)
		sources = append(sources, Source{
			Type:        t,
			Reference:   ref,
			Fingerprint: fp,
			Metadata:    metadata,
			AddedAt:     now,
		})
		sizeBytes += len(t) + len(ref) + len(fp)
	}

	provenance := "rule-based context builder; sources=" + util.Itoa(len(sources))
	return Context{
		RequestID:  requestID,
		Sources:    sources,
		Provenance: provenance,
		SizeBytes:  sizeBytes,
		BuiltAt:    now,
	}, nil
}

func fingerprint(t, ref string) string {
	var b strings.Builder
	b.WriteString(t)
	b.WriteByte(':')
	b.WriteString(ref)
	return b.String()
}
