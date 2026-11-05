package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
)

func TestProviderIDAndCapabilities(t *testing.T) {
	p := New("test-key")
	if p.ID() != "openai" {
		t.Fatalf("id=%q, want openai", p.ID())
	}
	if !p.Enabled() {
		t.Fatal("provider should be enabled with API key")
	}
	caps := p.Capabilities()
	if len(caps) == 0 {
		t.Fatal("capabilities should not be empty")
	}
}

func TestProviderDisabledWithoutKey(t *testing.T) {
	p := New("")
	if p.Enabled() {
		t.Fatal("provider should be disabled without API key")
	}
}

func TestGenerateDisabled(t *testing.T) {
	p := New("")
	_, err := p.Generate(context.Background(), model.Request{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected error for disabled provider")
	}
}

func TestGenerateEmptyPrompt(t *testing.T) {
	p := New("test-key")
	_, err := p.Generate(context.Background(), model.Request{Prompt: "   "})
	if err == nil {
		t.Fatal("expected error for empty prompt")
	}
}

func TestGenerateSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1234567890,
			"model": "gpt-3.5-turbo",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "Hello, world!"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL), WithModel("gpt-3.5-turbo"))
	resp, err := p.Generate(context.Background(), model.Request{Prompt: "Say hello"})
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	if resp.Text != "Hello, world!" {
		t.Fatalf("text=%q, want Hello, world!", resp.Text)
	}
	if resp.ModelID != "gpt-3.5-turbo" {
		t.Fatalf("model_id=%q, want gpt-3.5-turbo", resp.ModelID)
	}
}

func TestGenerateHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	p := New("bad-key", WithBaseURL(server.URL))
	_, err := p.Generate(context.Background(), model.Request{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("err=%v, want 401 in message", err)
	}
}

func TestHealthCheckSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	if err := p.HealthCheck(context.Background()); err != nil {
		t.Fatalf("health check error: %v", err)
	}
}

func TestHealthCheckDisabled(t *testing.T) {
	p := New("")
	if err := p.HealthCheck(context.Background()); err == nil {
		t.Fatal("expected error for disabled provider")
	}
}

func TestCustomHTTPClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		_, _ = w.Write([]byte(`{"choices": []}`))
	}))
	defer server.Close()

	shortClient := &http.Client{Timeout: 100 * time.Millisecond}
	p := New("test-key", WithBaseURL(server.URL), WithHTTPClient(shortClient))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := p.Generate(ctx, model.Request{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
