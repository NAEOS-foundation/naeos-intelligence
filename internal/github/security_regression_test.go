package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
)

func TestClient_GetRepository_ErrorBodyNotExposed(t *testing.T) {
	const secret = "upstream-sensitive-detail"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat(secret, 1024)))
	}))
	defer srv.Close()

	client := NewClient("token", "test")
	client.SetBaseURL(srv.URL)
	_, err := client.GetRepository(context.Background(), "owner", "repo")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error exposes upstream body: %v", err)
	}
	if !strings.Contains(err.Error(), "status=500") {
		t.Fatalf("error should include status: %v", err)
	}
}

func TestClient_GetRepository_UnauthorizedBodyNotExposed(t *testing.T) {
	const secret = "private-auth-diagnostic"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(secret))
	}))
	defer srv.Close()

	client := NewClient("bad-token", "test")
	client.SetBaseURL(srv.URL)
	_, err := client.GetRepository(context.Background(), "owner", "repo")
	if !errors.Is(err, errUnauthorized) {
		t.Fatalf("err=%v, want errUnauthorized", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error exposes upstream body: %v", err)
	}
}

func TestParseRepositoryRef_RejectsInvalidSegments(t *testing.T) {
	tests := []string{
		"", "owner", "/repo", "owner/", "owner/repo/extra",
		"../repo", "owner/..", "owner/.", "owner/re po",
		"owner/repo\\name", "owner/repo?query", "owner/repo\n",
	}
	for _, ref := range tests {
		t.Run(ref, func(t *testing.T) {
			if _, _, err := parseRepositoryRef(ref); err == nil {
				t.Fatalf("parseRepositoryRef(%q) should reject reference", ref)
			}
		})
	}
}

func TestParseRepositoryRef_AcceptsValidSegments(t *testing.T) {
	owner, name, err := parseRepositoryRef("NAEOS-foundation/naeos_intelligence.v2")
	if err != nil {
		t.Fatalf("parseRepositoryRef: %v", err)
	}
	if owner != "NAEOS-foundation" || name != "naeos_intelligence.v2" {
		t.Fatalf("got owner=%q name=%q", owner, name)
	}
}

func TestGitHubSourceResolver_InvalidReferenceDoesNotCallProvider(t *testing.T) {
	resolver := NewGitHubSourceResolver(nil)
	_, err := resolver.Resolve(context.Background(), intctx.SourceInput{
		Type: "repository", Reference: "../repo",
	})
	if err == nil {
		t.Fatal("expected invalid reference error")
	}
	if !strings.Contains(err.Error(), "owner/name") {
		t.Fatalf("err=%v, want owner/name guidance", err)
	}
}
