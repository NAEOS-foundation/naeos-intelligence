package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_GetRepository(t *testing.T) {
	repoJSON := `{"id":123,"name":"demo","full_name":"owner/demo","description":"A demo repository","private":false,"fork":false,"archived":false,"default_branch":"main","open_issues_count":5,"stargazers_count":42,"forks_count":7,"size":1024,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-02-01T00:00:00Z","pushed_at":"2024-03-01T00:00:00Z","html_url":"https://github.com/owner/demo"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/demo" {
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
	c.base = srv.URL

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
	c.base = srv.URL

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
	c.base = srv.URL

	_, err := c.GetRepository(context.Background(), "owner", "nonexistent")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if err.Error() != "github: resource not found" {
		t.Errorf("err=%v, want resource not found", err)
	}
}

func TestClient_ListIssues(t *testing.T) {
	issuesJSON := `[
		{"id":1,"number":101,"title":"Fix bug","body":"body","state":"open","author":"alice","labels":["bug"],"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z","html_url":"https://github.com/owner/demo/issues/101"},
		{"id":2,"number":102,"title":"Add feature","body":"body","state":"open","author":"bob","labels":["enhancement"],"created_at":"2024-01-03T00:00:00Z","updated_at":"2024-01-04T00:00:00Z","html_url":"https://github.com/owner/demo/issues/102"}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/demo/issues" {
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
	c.base = srv.URL

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

func TestClient_GetPullRequest(t *testing.T) {
	prJSON := `{"id":999,"number":5,"title":"Refactor API","body":"body","state":"open","author":"carol","base_branch":"main","head_branch":"feature/api","merged":false,"additions":100,"deletions":20,"changed_files":5,"created_at":"2024-02-01T00:00:00Z","updated_at":"2024-02-02T00:00:00Z","html_url":"https://github.com/owner/demo/pull/5"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/demo/pulls/5" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(prJSON))
	}))
	defer srv.Close()

	c := NewClient("test-token", "naeos-bot")
	c.base = srv.URL

	pr, err := c.GetPullRequest(context.Background(), "owner", "demo", 5)
	if err != nil {
		t.Fatalf("GetPullRequest: %v", err)
	}
	if pr.Number != 5 || pr.Title != "Refactor API" {
		t.Errorf("pr=%+v", pr)
	}
	if pr.BaseBranch != "main" || pr.HeadBranch != "feature/api" {
		t.Errorf("branches: base=%q head=%q", pr.BaseBranch, pr.HeadBranch)
	}
	if pr.Additions != 100 || pr.Deletions != 20 {
		t.Errorf("additions=%d deletions=%d", pr.Additions, pr.Deletions)
	}
}

func TestClient_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never respond; client must respect context cancellation.
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := NewClient("", "naeos-bot")
	c.base = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.GetRepository(ctx, "owner", "demo")
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}
