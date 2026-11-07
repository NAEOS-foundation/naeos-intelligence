package naeoscore

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
)

func TestClient_GetRepository(t *testing.T) {
	repoJSON := `{"id":"repo-123","name":"demo","full_name":"owner/demo","description":"A demo repository","private":false,"archived":false,"default_branch":"main","open_issues":5,"stars":42,"forks":7,"size_kb":1024,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-02-01T00:00:00Z","url":"https://naeos.dev/repos/owner/demo"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/repositories/owner/demo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected authorization: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(repoJSON))
	}))
	defer srv.Close()

	c := NewClient("test-token", "naeos-bot")
	c.SetBaseURL(srv.URL)

	repo, err := c.GetRepository(context.Background(), "owner", "demo")
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	if repo.FullName != "owner/demo" {
		t.Errorf("full_name=%q, want owner/demo", repo.FullName)
	}
	if repo.Owner != "owner" {
		t.Errorf("owner=%q, want owner", repo.Owner)
	}
	if repo.Stars != 42 {
		t.Errorf("stars=%d, want 42", repo.Stars)
	}
	if repo.CreatedAt.IsZero() {
		t.Error("created_at must not be zero")
	}
}

func TestClient_GetRepository_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	c := NewClient("bad-token", "naeos-bot")
	c.SetBaseURL(srv.URL)

	_, err := c.GetRepository(context.Background(), "owner", "demo")
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	if !errors.Is(err, errUnauthorized) {
		t.Errorf("err=%v, want errUnauthorized", err)
	}
}

func TestClient_GetRepository_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient("", "naeos-bot")
	c.SetBaseURL(srv.URL)

	_, err := c.GetRepository(context.Background(), "owner", "nonexistent")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if err.Error() != "naeoscore: resource not found" {
		t.Errorf("err=%v, want resource not found", err)
	}
}

func TestClient_ListIssues(t *testing.T) {
	issuesJSON := `[
		{"id":"issue-1","number":101,"title":"Fix bug","body":"body","state":"open","author":"alice","labels":["bug"],"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z","url":"https://naeos.dev/repos/owner/demo/issues/101"},
		{"id":"issue-2","number":102,"title":"Add feature","body":"body","state":"open","author":"bob","labels":["enhancement"],"created_at":"2024-01-03T00:00:00Z","updated_at":"2024-01-04T00:00:00Z","url":"https://naeos.dev/repos/owner/demo/issues/102"}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/repositories/owner/demo/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "state=open") {
			t.Errorf("missing state=open query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(issuesJSON))
	}))
	defer srv.Close()

	c := NewClient("test-token", "naeos-bot")
	c.SetBaseURL(srv.URL)

	issues, err := c.ListIssues(context.Background(), "owner", "demo", 10)
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 2 {
		t.Errorf("issues count=%d, want 2", len(issues))
	}
	if issues[0].Number != 101 || issues[0].Title != "Fix bug" {
		t.Errorf("issue[0]=%+v", issues[0])
	}
	if issues[1].Number != 102 || issues[1].Title != "Add feature" {
		t.Errorf("issue[1]=%+v", issues[1])
	}
}

func TestClient_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := NewClient("", "naeos-bot")
	c.SetBaseURL(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.GetRepository(ctx, "owner", "demo")
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}

func TestCoreSourceResolver_Resolve_NAEOSCore(t *testing.T) {
	repoJSON := `{"id":"repo-123","name":"demo","full_name":"owner/demo","description":"A demo","private":false,"archived":false,"default_branch":"main","open_issues":5,"stars":42,"forks":7,"size_kb":1024,"url":"https://naeos.dev/repos/owner/demo"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/repositories/owner/demo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(repoJSON))
	}))
	defer srv.Close()

	client := NewClient("token", "bot")
	client.SetBaseURL(srv.URL)
	resolver := NewCoreSourceResolver(client)

	resolved, err := resolver.Resolve(context.Background(), intctx.SourceInput{Type: "naeos-core", Reference: "owner/demo"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Type != "naeos-core" {
		t.Errorf("type=%q, want naeos-core", resolved.Type)
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

func TestCoreSourceResolver_Resolve_NonNAEOSCore(t *testing.T) {
	resolver := NewCoreSourceResolver(nil)

	resolved, err := resolver.Resolve(context.Background(), intctx.SourceInput{Type: "issue", Reference: "ISSUE-123"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Type != "issue" {
		t.Errorf("type=%q, want issue", resolved.Type)
	}
	if resolved.Metadata == nil {
		t.Error("metadata should be initialized")
	}
}

func TestCoreSourceResolver_Resolve_ValidationErrors(t *testing.T) {
	resolver := NewCoreSourceResolver(nil)

	tests := []struct {
		name    string
		input   intctx.SourceInput
		wantErr string
	}{
		{"empty type", intctx.SourceInput{Type: "", Reference: "owner/demo"}, "source type is required"},
		{"empty reference", intctx.SourceInput{Type: "naeos-core", Reference: ""}, "source reference is required"},
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

func TestCoreSourceResolver_Resolve_InvalidRef(t *testing.T) {
	resolver := NewCoreSourceResolver(nil)

	_, err := resolver.Resolve(context.Background(), intctx.SourceInput{Type: "naeos-core", Reference: "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid ref")
	}
	if !strings.Contains(err.Error(), "owner/name") {
		t.Errorf("err=%v, want owner/name hint", err)
	}
}

func TestCoreSourceResolver_Resolve_CoreError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewClient("", "bot")
	client.SetBaseURL(srv.URL)
	resolver := NewCoreSourceResolver(client)

	_, err := resolver.Resolve(context.Background(), intctx.SourceInput{Type: "naeos-core", Reference: "owner/nonexistent"})
	if err == nil {
		t.Fatal("expected error from NAEOS Core API")
	}
	if !strings.Contains(err.Error(), "resource not found") {
		t.Errorf("err=%v, want resource not found", err)
	}
}
