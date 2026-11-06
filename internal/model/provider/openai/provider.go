// Package openai provides an OpenAI-compatible chat completions model provider adapter.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-3.5-turbo"
	chatEndpoint   = "/chat/completions"
)

// Provider is an OpenAI-compatible model provider adapter.
type Provider struct {
	client  *http.Client
	baseURL string
	apiKey  string
	model   string
	enabled bool
	caps    []string
}

// chatRequest is the OpenAI chat completions request body.
type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

// message is a single chat message.
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the OpenAI chat completions response body.
type chatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

// choice is a single completion choice.
type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// usage is token usage information.
type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Option configures a Provider.
type Option func(*Provider)

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(p *Provider) {
		p.baseURL = strings.TrimRight(url, "/")
	}
}

// WithModel sets the default model ID.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.model = model
	}
}

// WithCapabilities sets the advertised capabilities.
func WithCapabilities(caps ...string) Option {
	return func(p *Provider) {
		p.caps = caps
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		p.client = client
	}
}

// Enabled returns whether the provider is enabled.
func (p *Provider) Enabled() bool { return p.enabled }

// ID returns the provider identifier.
func (p *Provider) ID() string { return "openai" }

// New creates an OpenAI-compatible provider. The provider is disabled by default;
// call WithEnabled(true) or set the API key explicitly to enable it.
func New(apiKey string, opts ...Option) *Provider {
	p := &Provider{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		model:   defaultModel,
		enabled: apiKey != "",
		caps:    []string{"chat", "completion"},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithEnabled sets whether the provider is enabled.
func WithEnabled(enabled bool) Option {
	return func(p *Provider) {
		p.enabled = enabled
	}
}

// Generate sends a chat completion request to the provider.
func (p *Provider) Generate(ctx context.Context, req model.Request) (model.Response, error) {
	if !p.enabled {
		return model.Response{}, errors.New("provider is disabled")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return model.Response{}, errors.New("prompt is required")
	}

	modelID := req.ModelID
	if modelID == "" {
		modelID = p.model
	}

	body := chatRequest{
		Model:    modelID,
		Messages: []message{{Role: "user", Content: req.Prompt}},
		Stream:   false,
	}
	if v, ok := req.Metadata["temperature"]; ok {
		fmt.Sscanf(v, "%f", &body.Temperature)
	}
	if v, ok := req.Metadata["max_tokens"]; ok {
		fmt.Sscanf(v, "%d", &body.MaxTokens)
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return model.Response{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+chatEndpoint, bytes.NewReader(payload))
	if err != nil {
		return model.Response{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return model.Response{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return model.Response{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.Response{}, fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return model.Response{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return model.Response{}, errors.New("provider returned no choices")
	}

	return model.Response{
		ModelID: chatResp.Model,
		Text:    chatResp.Choices[0].Message.Content,
	}, nil
}

// HealthCheck checks provider connectivity by listing models.
func (p *Provider) HealthCheck(ctx context.Context) error {
	if !p.enabled {
		return errors.New("provider is disabled")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("provider returned status %d", resp.StatusCode)
	}
	return nil
}

// Capabilities returns the provider's supported capabilities.
func (p *Provider) Capabilities() []string {
	caps := make([]string, len(p.caps))
	copy(caps, p.caps)
	return caps
}
