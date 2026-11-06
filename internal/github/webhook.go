package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// errInvalidSignature is returned when the webhook signature does not match.
var errInvalidSignature = errors.New("github: invalid webhook signature")

// errMissingSignature is returned when the X-Hub-Signature-256 header is absent.
var errMissingSignature = errors.New("github: missing X-Hub-Signature-256 header")

// errMissingEvent is returned when the X-GitHub-Event header is absent.
var errMissingEvent = errors.New("github: missing X-GitHub-Event header")

// WebhookValidator validates GitHub webhook signatures and extracts the event type.
type WebhookValidator struct {
	secret []byte
}

// NewWebhookValidator creates a validator with the GitHub webhook secret.
func NewWebhookValidator(secret string) *WebhookValidator {
	return &WebhookValidator{secret: []byte(secret)}
}

// Validate checks the signature header against the body and returns the event type.
// The body is consumed and returned so callers can re-read it.
func (v *WebhookValidator) Validate(r *http.Request) (eventType string, body []byte, err error) {
	if r.Body == nil {
		return "", nil, errors.New("github: request body is nil")
	}
	body, err = io.ReadAll(r.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read webhook body: %w", err)
	}

	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return "", nil, errMissingSignature
	}

	eventType = r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		return "", nil, errMissingEvent
	}

	if !v.validSignature(string(body), signature) {
		return "", nil, errInvalidSignature
	}

	return eventType, body, nil
}

func (v *WebhookValidator) validSignature(body, signature string) bool {
	expected := hex.EncodeToString(hmacSha256(v.secret, []byte(body)))
	provided := strings.TrimPrefix(signature, "sha256=")
	return hmac.Equal([]byte(expected), []byte(provided))
}

func hmacSha256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(data)
	return h.Sum(nil)
}

// EventType is a recognized GitHub webhook event.
type EventType string

const (
	EventPush              EventType = "push"
	EventPullRequest       EventType = "pull_request"
	EventPullRequestReview EventType = "pull_request_review"
	EventIssues            EventType = "issues"
	EventIssueComment      EventType = "issue_comment"
	EventCreate            EventType = "create"
	EventDelete            EventType = "delete"
	EventRepository        EventType = "repository"
)

// IsSupported reports whether the event type is processed by the intelligence pipeline.
func (e EventType) IsSupported() bool {
	switch e {
	case EventPush, EventPullRequest, EventPullRequestReview, EventIssues, EventIssueComment, EventCreate, EventDelete, EventRepository:
		return true
	default:
		return false
	}
}

// WebhookEvent is a parsed, validated GitHub webhook event.
type WebhookEvent struct {
	ID         string         `json:"id"`
	Type       EventType      `json:"type"`
	Repository Repository     `json:"repository"`
	Sender     string         `json:"sender"`
	Action     string         `json:"action,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
	ReceivedAt time.Time      `json:"received_at"`
}

// ParseEvent parses a validated webhook body into a WebhookEvent.
func ParseEvent(body []byte, eventType string) (*WebhookEvent, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse webhook payload: %w", err)
	}

	evt := &WebhookEvent{
		Type:       EventType(eventType),
		ReceivedAt: time.Now().UTC(),
		Payload:    raw,
	}

	if id, ok := raw["id"]; ok {
		if s, ok := id.(string); ok {
			evt.ID = s
		}
	}
	if repo, ok := raw["repository"].(map[string]any); ok {
		evt.Repository = mapRepository(repo)
	}
	if sender, ok := raw["sender"].(map[string]any); ok {
		if login, ok := sender["login"].(string); ok {
			evt.Sender = login
		}
	}
	if action, ok := raw["action"].(string); ok {
		evt.Action = action
	}

	return evt, nil
}

func mapRepository(raw map[string]any) Repository {
	repo := Repository{}
	if v, ok := raw["id"].(float64); ok {
		repo.ID = int64(v)
	}
	if v, ok := raw["name"].(string); ok {
		repo.Name = v
	}
	if v, ok := raw["full_name"].(string); ok {
		repo.FullName = v
	}
	if v, ok := raw["description"].(string); ok {
		repo.Description = v
	}
	if v, ok := raw["private"].(bool); ok {
		repo.Private = v
	}
	if v, ok := raw["fork"].(bool); ok {
		repo.Fork = v
	}
	if v, ok := raw["archived"].(bool); ok {
		repo.Archived = v
	}
	if v, ok := raw["default_branch"].(string); ok {
		repo.DefaultBranch = v
	}
	if v, ok := raw["open_issues_count"].(float64); ok {
		repo.OpenIssues = int(v)
	}
	if v, ok := raw["stargazers_count"].(float64); ok {
		repo.Stars = int(v)
	}
	if v, ok := raw["forks_count"].(float64); ok {
		repo.Forks = int(v)
	}
	if v, ok := raw["size"].(float64); ok {
		repo.SizeKB = int(v)
	}
	if v, ok := raw["html_url"].(string); ok {
		repo.URL = v
	}
	if owner, ok := raw["owner"].(map[string]any); ok {
		if login, ok := owner["login"].(string); ok {
			repo.Owner = login
		}
	}
	return repo
}
