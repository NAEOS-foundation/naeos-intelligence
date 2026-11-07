package context

import (
	"context"
	"errors"
	"testing"
)

func TestRuleBuilder_Build(t *testing.T) {
	b := NewRuleBuilder()

	tests := []struct {
		name      string
		requestID string
		inputs    []SourceInput
		wantErr   bool
		wantN     int
	}{
		{
			name:      "empty request id returns error",
			requestID: "   ",
			wantErr:   true,
		},
		{
			name:      "valid repository source",
			requestID: "req-1",
			inputs:    []SourceInput{{Type: "repository", Reference: "github.com/owner/repo"}},
			wantN:     1,
		},
		{
			name:      "valid issue source",
			requestID: "req-2",
			inputs:    []SourceInput{{Type: "issue", Reference: "ISSUE-123"}},
			wantN:     1,
		},
		{
			name:      "valid pull request source",
			requestID: "req-3",
			inputs:    []SourceInput{{Type: "pull_request", Reference: "github.com/owner/repo/pull/42"}},
			wantN:     1,
		},
		{
			name:      "valid documentation source",
			requestID: "req-4",
			inputs:    []SourceInput{{Type: "documentation", Reference: "docs/api.md"}},
			wantN:     1,
		},
		{
			name:      "valid commit source",
			requestID: "req-5",
			inputs:    []SourceInput{{Type: "commit", Reference: "abc123def456"}},
			wantN:     1,
		},
		{
			name:      "valid naeos-core source",
			requestID: "req-6",
			inputs:    []SourceInput{{Type: "naeos-core", Reference: "naeos-core/repo"}},
			wantN:     1,
		},
		{
			name:      "unsupported source type returns error",
			requestID: "req-7",
			inputs:    []SourceInput{{Type: "unknown_type", Reference: "ref"}},
			wantErr:   true,
		},
		{
			name:      "empty reference returns error",
			requestID: "req-8",
			inputs:    []SourceInput{{Type: "repository", Reference: "   "}},
			wantErr:   true,
		},
		{
			name:      "too many sources returns error",
			requestID: "req-9",
			inputs:    makeSources(101),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			res, err := b.Build(ctx, tt.requestID, tt.inputs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.RequestID != tt.requestID {
				t.Errorf("request_id = %q, want %q", res.RequestID, tt.requestID)
			}
			if len(res.Sources) != tt.wantN {
				t.Errorf("sources count = %d, want %d", len(res.Sources), tt.wantN)
			}
			if res.Provenance == "" {
				t.Error("provenance must not be empty")
			}
			if res.BuiltAt.IsZero() {
				t.Error("built_at must not be zero")
			}
			for _, s := range res.Sources {
				if s.Fingerprint == "" {
					t.Error("fingerprint must not be empty")
				}
				if s.AddedAt.IsZero() {
					t.Error("added_at must not be zero")
				}
			}
		})
	}
}

func TestRuleBuilder_Build_ContextCancelled(t *testing.T) {
	b := NewRuleBuilder()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := b.Build(ctx, "req-1", []SourceInput{{Type: "repository", Reference: "repo"}})
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}

func makeSources(n int) []SourceInput {
	s := make([]SourceInput, n)
	for i := range s {
		s[i] = SourceInput{Type: "repository", Reference: "repo"}
	}
	return s
}

type fakeResolver struct {
	resolved ResolvedSource
	err      error
}

func (f *fakeResolver) Resolve(ctx context.Context, input SourceInput) (ResolvedSource, error) {
	if f.err != nil {
		return ResolvedSource{}, f.err
	}
	return f.resolved, nil
}

func TestRuleBuilder_Build_WithResolver(t *testing.T) {
	resolved := ResolvedSource{
		Type:      "repository",
		Reference: "owner/demo",
		Metadata: map[string]any{
			"full_name": "owner/demo",
			"stars":     42,
		},
	}
	b := NewRuleBuilder(WithResolver(&fakeResolver{resolved: resolved}))

	ctx := context.Background()
	res, err := b.Build(ctx, "req-1", []SourceInput{{Type: "repository", Reference: "owner/demo"}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(res.Sources) != 1 {
		t.Fatalf("sources count=%d, want 1", len(res.Sources))
	}
	src := res.Sources[0]
	if src.Metadata["full_name"] != "owner/demo" {
		t.Errorf("full_name=%v, want owner/demo", src.Metadata["full_name"])
	}
	if src.Metadata["stars"] != 42 {
		t.Errorf("stars=%v, want 42", src.Metadata["stars"])
	}
}

func TestRuleBuilder_Build_ResolverError(t *testing.T) {
	b := NewRuleBuilder(WithResolver(&fakeResolver{err: errors.New("resolver failed")}))

	_, err := b.Build(context.Background(), "req-1", []SourceInput{{Type: "repository", Reference: "owner/demo"}})
	if err == nil {
		t.Fatal("expected resolver error")
	}
	if err.Error() != "resolve source repository:owner/demo: resolver failed" {
		t.Errorf("err=%v, want wrapped resolver error", err)
	}
}
