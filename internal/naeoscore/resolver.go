package naeoscore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
)

// CoreSourceResolver resolves naeos-core-type sources using the NAEOS Core API.
type CoreSourceResolver struct {
	client *Client
}

// NewCoreSourceResolver creates a resolver backed by a NAEOS Core client.
func NewCoreSourceResolver(client *Client) *CoreSourceResolver {
	return &CoreSourceResolver{client: client}
}

// Resolve fetches repository metadata from NAEOS Core for naeos-core-type sources.
// Implements intctx.SourceResolver.
func (r *CoreSourceResolver) Resolve(ctx context.Context, input intctx.SourceInput) (intctx.ResolvedSource, error) {
	if strings.TrimSpace(input.Type) == "" {
		return intctx.ResolvedSource{}, errors.New("source type is required")
	}
	if strings.TrimSpace(input.Reference) == "" {
		return intctx.ResolvedSource{}, errors.New("source reference is required")
	}

	if input.Type != "naeos-core" {
		return intctx.ResolvedSource{
			Type:      input.Type,
			Reference: input.Reference,
			Metadata:  map[string]any{},
		}, nil
	}

	owner, name, err := parseCoreRef(input.Reference)
	if err != nil {
		return intctx.ResolvedSource{}, err
	}

	repo, err := r.client.GetRepository(ctx, owner, name)
	if err != nil {
		return intctx.ResolvedSource{}, fmt.Errorf("resolve naeos-core repository %s/%s: %w", owner, name, err)
	}

	return intctx.ResolvedSource{
		Type:      "naeos-core",
		Reference: input.Reference,
		Metadata: map[string]any{
			"full_name":      repo.FullName,
			"description":    repo.Description,
			"private":        repo.Private,
			"archived":       repo.Archived,
			"default_branch": repo.DefaultBranch,
			"open_issues":    repo.OpenIssues,
			"stars":          repo.Stars,
			"forks":          repo.Forks,
			"size_kb":        repo.SizeKB,
			"url":            repo.URL,
		},
	}, nil
}

func parseCoreRef(ref string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(ref), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid naeos-core reference %q, want owner/name", ref)
	}
	return parts[0], parts[1], nil
}
