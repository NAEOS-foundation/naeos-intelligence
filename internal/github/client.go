package github

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

// Client is a minimal, authenticated GitHub REST API client.
type Client struct {
	http  *http.Client
	base  string
	token string
	user  string
}

// NewClient creates a GitHub API client. token may be empty for unauthenticated
// (rate-limited) access; user is the GitHub login used for audit logging.
func NewClient(token, user string) *Client {
	return &Client{
		http:  &http.Client{Timeout: 30 * time.Second},
		base:  "https://api.github.com",
		token: strings.TrimSpace(token),
		user:  strings.TrimSpace(user),
	}
}

// SetBaseURL overrides the GitHub API base URL (used for testing).
func (c *Client) SetBaseURL(url string) {
	c.base = url
}

// Repository is a subset of the GitHub repository resource relevant to
// engineering intelligence. Read-only; no fields are mutated.
type Repository struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Owner         string    `json:"owner"`
	Description   string    `json:"description"`
	Private       bool      `json:"private"`
	Fork          bool      `json:"fork"`
	Archived      bool      `json:"archived"`
	DefaultBranch string    `json:"default_branch"`
	OpenIssues    int       `json:"open_issues_count"`
	Stars         int       `json:"stargazers_count"`
	Forks         int       `json:"forks_count"`
	SizeKB        int       `json:"size"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	PushedAt      time.Time `json:"pushed_at"`
	URL           string    `json:"html_url"`
}

// Issue is a subset of the GitHub issue resource.
type Issue struct {
	ID        int64      `json:"id"`
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
	URL       string     `json:"html_url"`
}

// PullRequest is a subset of the GitHub pull request resource.
type PullRequest struct {
	ID           int64      `json:"id"`
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	State        string     `json:"state"`
	Author       string     `json:"author"`
	BaseBranch   string     `json:"base_branch"`
	HeadBranch   string     `json:"head_branch"`
	Merged       bool       `json:"merged"`
	Mergeable    *bool      `json:"mergeable"`
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
	ChangedFiles int        `json:"changed_files"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ClosedAt     *time.Time `json:"closed_at"`
	URL          string     `json:"html_url"`
}

// GetRepository fetches a repository by owner/name.
func (c *Client) GetRepository(ctx context.Context, owner, name string) (*Repository, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, name)
	var repo Repository
	if err := c.get(ctx, path, &repo); err != nil {
		return nil, err
	}
	repo.Owner = owner
	return &repo, nil
}

// ListIssues fetches open issues for a repository (bounded to maxResults).
func (c *Client) ListIssues(ctx context.Context, owner, name string, maxResults int) ([]Issue, error) {
	if maxResults <= 0 || maxResults > 100 {
		maxResults = 30
	}
	path := fmt.Sprintf("/repos/%s/%s/issues?state=open&per_page=%d", owner, name, maxResults)
	var issues []Issue
	if err := c.get(ctx, path, &issues); err != nil {
		return nil, err
	}
	for i := range issues {
		issues[i].Author = c.extractLogin(issues[i].Author)
	}
	return issues, nil
}

// GetPullRequest fetches a pull request by number.
func (c *Client) GetPullRequest(ctx context.Context, owner, name string, number int) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, name, number)
	var pr PullRequest
	if err := c.get(ctx, path, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// errUnauthorized is returned when the GitHub token is invalid or missing.
var errUnauthorized = errors.New("github: unauthorized or rate-limited")

func (c *Client) get(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
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
		return fmt.Errorf("github request %s: %w", path, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		// Do not consume or expose provider error bodies: they may be large or
		// contain sensitive details. The deferred close releases the response.
		return fmt.Errorf("%w (status=%d)", errUnauthorized, resp.StatusCode)
	case http.StatusNotFound:
		return errors.New("github: resource not found")
	case http.StatusTooManyRequests:
		return errors.New("github: rate limit exceeded")
	default:
		// Error bodies are intentionally not read; keep memory use bounded and
		// avoid leaking upstream response content into application errors.
		return fmt.Errorf("github request %s failed: status=%d", path, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return fmt.Errorf("read github response %s: %w", path, err)
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("decode github response %s: %w", path, err)
	}
	return nil
}

func (c *Client) extractLogin(s string) string {
	// Author field may already be a login string from our simplified mapping.
	return strings.TrimSpace(s)
}
