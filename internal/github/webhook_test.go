package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhookValidator_Validate_Valid(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"repository":{"full_name":"owner/demo"},"sender":{"login":"alice"}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(string(body)))
	req.Header.Set("X-Hub-Signature-256", signature)
	req.Header.Set("X-GitHub-Event", "push")

	v := NewWebhookValidator(secret)
	eventType, gotBody, err := v.Validate(req)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if eventType != "push" {
		t.Errorf("eventType=%q, want push", eventType)
	}
	if string(gotBody) != string(body) {
		t.Error("body mismatch")
	}
}

func TestWebhookValidator_Validate_MissingSignature(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(`{}`))
	req.Header.Set("X-GitHub-Event", "push")

	v := NewWebhookValidator("secret")
	_, _, err := v.Validate(req)
	if err == nil {
		t.Fatal("expected missing signature error")
	}
	if err.Error() != "github: missing X-Hub-Signature-256 header" {
		t.Errorf("err=%v", err)
	}
}

func TestWebhookValidator_Validate_MissingEvent(t *testing.T) {
	body := []byte(`{}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(string(body)))
	req.Header.Set("X-Hub-Signature-256", signature)

	v := NewWebhookValidator("secret")
	_, _, err := v.Validate(req)
	if err == nil {
		t.Fatal("expected missing event error")
	}
	if err.Error() != "github: missing X-GitHub-Event header" {
		t.Errorf("err=%v", err)
	}
}

func TestWebhookValidator_Validate_InvalidSignature(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(`{}`))
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")
	req.Header.Set("X-GitHub-Event", "push")

	v := NewWebhookValidator("secret")
	_, _, err := v.Validate(req)
	if err == nil {
		t.Fatal("expected invalid signature error")
	}
	if err.Error() != "github: invalid webhook signature" {
		t.Errorf("err=%v", err)
	}
}

func TestParseEvent(t *testing.T) {
	body := []byte(`{"id":"evt_123","repository":{"full_name":"owner/demo","name":"demo","owner":{"login":"owner"}},"sender":{"login":"alice"},"action":"opened"}`)
	evt, err := ParseEvent(body, "pull_request")
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if evt.ID != "evt_123" {
		t.Errorf("id=%q, want evt_123", evt.ID)
	}
	if evt.Type != "pull_request" {
		t.Errorf("type=%q, want pull_request", evt.Type)
	}
	if evt.Repository.FullName != "owner/demo" {
		t.Errorf("repo=%q, want owner/demo", evt.Repository.FullName)
	}
	if evt.Sender != "alice" {
		t.Errorf("sender=%q, want alice", evt.Sender)
	}
	if evt.Action != "opened" {
		t.Errorf("action=%q, want opened", evt.Action)
	}
	if evt.ReceivedAt.IsZero() {
		t.Error("received_at must not be zero")
	}
}

func TestParseEvent_InvalidJSON(t *testing.T) {
	_, err := ParseEvent([]byte(`not-json`), "push")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestEventType_IsSupported(t *testing.T) {
	supported := []EventType{"push", "pull_request", "pull_request_review", "issues", "issue_comment", "create", "delete", "repository"}
	for _, e := range supported {
		if !e.IsSupported() {
			t.Errorf("%q should be supported", e)
		}
	}
	if EventType("unknown").IsSupported() {
		t.Error("unknown should not be supported")
	}
}

func TestWebhookValidator_Validate_NilBody(t *testing.T) {
	req := &http.Request{Header: make(http.Header)}
	req.Header.Set("X-Hub-Signature-256", "sha256=abc")
	req.Header.Set("X-GitHub-Event", "push")

	v := NewWebhookValidator("secret")
	_, _, err := v.Validate(req)
	if err == nil {
		t.Fatal("expected nil body error")
	}
}

func TestWebhookValidator_Validate_EmptySecretFailsClosed(t *testing.T) {
	body := []byte(`{}`)
	mac := hmac.New(sha256.New, nil)
	_, _ = mac.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(string(body)))
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-GitHub-Event", "push")

	_, _, err := NewWebhookValidator("").Validate(req)
	if err == nil || err.Error() != "github: webhook secret is not configured" {
		t.Fatalf("err=%v, want missing-secret error", err)
	}
}

func TestWebhookValidator_Validate_OversizedPayloadRejected(t *testing.T) {
	body := strings.Repeat("a", maxWebhookPayloadBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256=unused")
	req.Header.Set("X-GitHub-Event", "push")

	_, _, err := NewWebhookValidator("secret").Validate(req)
	if err == nil || err.Error() != "github: webhook payload exceeds 1 MiB" {
		t.Fatalf("err=%v, want payload-too-large error", err)
	}
}
