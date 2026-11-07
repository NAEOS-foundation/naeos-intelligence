package github

import (
	"context"
	"errors"
	"fmt"
	"strings"

	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
)

// GitHubSourceResolver resolves repository-type sources using the GitHub API.
type GitHubSourceResolver struct {
	client *Client
}

// NewGitHubSourceResolver creates a resolver backed by a GitHub client.
func NewGitHubSourceResolver(client *Client) *GitHubSourceResolver {
	return &GitHubSourceResolver{client: client}
}

// Resolve fetches repository metadata from GitHub for repository-type sources.
// Implements intctx.SourceResolver.
func (r *GitHubSourceResolver) Resolve(ctx context.Context, input intctx.SourceInput) (intctx.ResolvedSource, error) {
	if strings.TrimSpace(input.Type) == "" {
		return intctx.ResolvedSource{}, errors.New("source type is required")
	}
	if strings.TrimSpace(input.Reference) == "" {
		return intctx.ResolvedSource{}, errors.New("source reference is required")
	}

	if input.Type != "repository" {
		return intctx.ResolvedSource{
			Type:      input.Type,
			Reference: input.Reference,
			Metadata:  map[string]any{},
		}, nil
	}

	owner, name, err := parseRepositoryRef(input.Reference)
	if err != nil {
		return intctx.ResolvedSource{}, err
	}

	repo, err := r.client.GetRepository(ctx, owner, name)
	if err != nil {
		return intctx.ResolvedSource{}, fmt.Errorf("resolve repository %s/%s: %w", owner, name, err)
	}

	return intctx.ResolvedSource{
		Type:      "repository",
		Reference: input.Reference,
		Metadata: map[string]any{
			"full_name":      repo.FullName,
			"description":    repo.Description,
			"private":        repo.Private,
			"fork":           repo.Fork,
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

func parseRepositoryRef(ref string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(ref), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository reference %q, want owner/name", ref)
	}
	return parts[0], parts[1], nil
}
