package naeoscore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a minimal authenticated client for the NAEOS Core API.
// It provides read-only access to engineering context exposed by NAEOS Core.
type Client struct {
	http  *http.Client
	base  string
	token string
	user  string
}

// NewClient creates a NAEOS Core API client. token may be empty for
// unauthenticated access; user is the client identity used for audit logging.
func NewClient(token, user string) *Client {
	return &Client{
		http:  &http.Client{Timeout: 30 * time.Second},
		base:  "https://api.naeos.dev",
		token: strings.TrimSpace(token),
		user:  strings.TrimSpace(user),
	}
}

// SetBaseURL overrides the NAEOS Core API base URL (used for testing).
func (c *Client) SetBaseURL(url string) {
	c.base = url
}

// Repository is engineering context retrieved from NAEOS Core.
type Repository struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Owner         string    `json:"owner"`
	Description   string    `json:"description"`
	Private       bool      `json:"private"`
	Archived      bool      `json:"archived"`
	DefaultBranch string    `json:"default_branch"`
	OpenIssues    int       `json:"open_issues"`
	Stars         int       `json:"stars"`
	Forks         int       `json:"forks"`
	SizeKB        int       `json:"size_kb"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	URL           string    `json:"url"`
}

// Issue is an engineering issue tracked in NAEOS Core.
type Issue struct {
	ID        string     `json:"id"`
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	Author    string     `json:"author"`
	Assignee  string     `json:"assignee"`
	Labels    []string   `json:"labels"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
	URL       string     `json:"url"`
}

// GetRepository fetches repository context from NAEOS Core.
func (c *Client) GetRepository(ctx context.Context, owner, name string) (*Repository, error) {
	path := fmt.Sprintf("/v1/repositories/%s/%s", owner, name)
	var repo Repository
	if err := c.get(ctx, path, &repo); err != nil {
		return nil, err
	}
	repo.Owner = owner
	return &repo, nil
}

// ListIssues fetches open issues for a repository from NAEOS Core.
func (c *Client) ListIssues(ctx context.Context, owner, name string, maxResults int) ([]Issue, error) {
	if maxResults <= 0 || maxResults > 100 {
		maxResults = 30
	}
	path := fmt.Sprintf("/v1/repositories/%s/%s/issues?state=open&limit=%d", owner, name, maxResults)
	var issues []Issue
	if err := c.get(ctx, path, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// errUnauthorized is returned when the NAEOS Core token is invalid or missing.
var errUnauthorized = errors.New("naeoscore: unauthorized or rate-limited")

func (c *Client) get(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-NAEOS-Api-Version", "2026-11-01")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.user != "" {
		req.Header.Set("User-Agent", "naeos-intelligence/"+c.user)
	} else {
		req.Header.Set("User-Agent", "naeos-intelligence")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("naeoscore request %s: %w", path, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w (status=%d body=%s)", errUnauthorized, resp.StatusCode, strings.TrimSpace(string(body)))
	case http.StatusNotFound:
		return errors.New("naeoscore: resource not found")
	case http.StatusTooManyRequests:
		return errors.New("naeoscore: rate limit exceeded")
	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("naeoscore request %s failed: status=%d body=%s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return fmt.Errorf("read naeoscore response %s: %w", path, err)
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("decode naeoscore response %s: %w", path, err)
	}
	return nil
}
