package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
)

func TestGitHubSourceResolver_Resolve_Repository(t *testing.T) {
	repoJSON := `{"id":123,"name":"demo","full_name":"owner/demo","description":"A demo","private":false,"fork":false,"archived":false,"default_branch":"main","open_issues_count":5,"stargazers_count":42,"forks_count":7,"size":1024,"html_url":"https://github.com/owner/demo"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/demo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(repoJSON))
	}))
	defer srv.Close()

	client := NewClient("token", "bot")
	client.SetBaseURL(srv.URL)
	resolver := NewGitHubSourceResolver(client)

	resolved, err := resolver.Resolve(context.Background(), intctx.SourceInput{Type: "repository", Reference: "owner/demo"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Type != "repository" {
		t.Errorf("type=%q, want repository", resolved.Type)
	}
	if resolved.Reference != "owner/demo" {
		t.Errorf("reference=%q, want owner/demo", resolved.Reference)
	}
	if resolved.Metadata["full_name"] != "owner/demo" {
		t.Errorf("full_name=%v, want owner/demo", resolved.Metadata["full_name"])
	}
	stars, ok := resolved.Metadata["stars"].(int)
	if !ok || stars != 42 {
		t.Errorf("stars=%v (%T), want int 42", resolved.Metadata["stars"], resolved.Metadata["stars"])
	}
}

func TestGitHubSourceResolver_Resolve_NonRepository(t *testing.T) {
	resolver := NewGitHubSourceResolver(nil)

	resolved, err := resolver.Resolve(context.Background(), intctx.SourceInput{Type: "issue", Reference: "ISSUE-123"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Type != "issue" {
		t.Errorf("type=%q, want issue", resolved.Type)
	}
	if resolved.Reference != "ISSUE-123" {
		t.Errorf("reference=%q, want ISSUE-123", resolved.Reference)
	}
	if resolved.Metadata == nil {
		t.Error("metadata should be initialized")
	}
}

func TestGitHubSourceResolver_Resolve_ValidationErrors(t *testing.T) {
	resolver := NewGitHubSourceResolver(nil)

	tests := []struct {
		name    string
		input   intctx.SourceInput
		wantErr string
	}{
		{"empty type", intctx.SourceInput{Type: "", Reference: "owner/demo"}, "source type is required"},
		{"empty reference", intctx.SourceInput{Type: "repository", Reference: ""}, "source reference is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolver.Resolve(context.Background(), tt.input)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("err=%v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGitHubSourceResolver_Resolve_InvalidRef(t *testing.T) {
	resolver := NewGitHubSourceResolver(nil)

	_, err := resolver.Resolve(context.Background(), intctx.SourceInput{Type: "repository", Reference: "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid ref")
	}
	if !strings.Contains(err.Error(), "owner/name") {
		t.Errorf("err=%v, want owner/name hint", err)
	}
}

func TestGitHubSourceResolver_Resolve_GitHubError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewClient("", "bot")
	client.SetBaseURL(srv.URL)
	resolver := NewGitHubSourceResolver(client)

	_, err := resolver.Resolve(context.Background(), intctx.SourceInput{Type: "repository", Reference: "owner/nonexistent"})
	if err == nil {
		t.Fatal("expected error from GitHub API")
	}
	if !strings.Contains(err.Error(), "resource not found") {
		t.Errorf("err=%v, want resource not found", err)
	}
}
