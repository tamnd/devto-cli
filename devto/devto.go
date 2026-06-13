// Package devto is the library behind the devto command line:
// the HTTP client, request shaping, and the typed data models for DEV
// Community (dev.to).
//
// The API is open and requires no authentication or API key. It is subject to
// per-IP rate limits; the client paces requests and retries 429/5xx with
// exponential backoff.
package devto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const apiBase = "https://dev.to/api"

// DefaultUserAgent identifies the client to the DEV API.
const DefaultUserAgent = "devto/dev (+https://github.com/tamnd/devto-cli)"

// ErrNotFound is returned when the DEV API reports a 404.
var ErrNotFound = errors.New("not found")

// Config holds constructor parameters for Client.
type Config struct {
	UserAgent string
	Rate      time.Duration
	Retries   int
	Workers   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
		Workers:   4,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the DEV Community API.
type Client struct {
	httpClient *http.Client
	userAgent  string
	rate       time.Duration
	retries    int
	workers    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
		workers:    cfg.Workers,
	}
}

// Articles fetches top articles. top is the number of days (7=week, 30=month,
// 365=year, 0=all-time). limit caps the number of results (max 1000).
func (c *Client) Articles(ctx context.Context, top int, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 1000 {
		limit = 1000
	}
	params := url.Values{}
	params.Set("top", strconv.Itoa(top))
	params.Set("per_page", strconv.Itoa(limit))
	rawURL := apiBase + "/articles?" + params.Encode()
	var wire []wireArticle
	if err := c.getJSON(ctx, rawURL, &wire); err != nil {
		return nil, err
	}
	return convertArticles(wire), nil
}

// TagArticles fetches articles tagged with tag, newest first.
func (c *Client) TagArticles(ctx context.Context, tag string, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 1000 {
		limit = 1000
	}
	params := url.Values{}
	params.Set("tag", tag)
	params.Set("per_page", strconv.Itoa(limit))
	rawURL := apiBase + "/articles?" + params.Encode()
	var wire []wireArticle
	if err := c.getJSON(ctx, rawURL, &wire); err != nil {
		return nil, err
	}
	return convertArticles(wire), nil
}

// UserArticles fetches articles published by username, newest first.
func (c *Client) UserArticles(ctx context.Context, username string, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 1000 {
		limit = 1000
	}
	params := url.Values{}
	params.Set("username", username)
	params.Set("per_page", strconv.Itoa(limit))
	rawURL := apiBase + "/articles?" + params.Encode()
	var wire []wireArticle
	if err := c.getJSON(ctx, rawURL, &wire); err != nil {
		return nil, err
	}
	return convertArticles(wire), nil
}

// ArticleByID fetches a single article by its integer id.
func (c *Client) ArticleByID(ctx context.Context, id int) (Article, error) {
	rawURL := fmt.Sprintf("%s/articles/%d", apiBase, id)
	var wire wireArticle
	if err := c.getJSON(ctx, rawURL, &wire); err != nil {
		return Article{}, err
	}
	return wireArticleToArticle(wire), nil
}

// User fetches a user profile by username.
func (c *Client) User(ctx context.Context, username string) (User, error) {
	params := url.Values{}
	params.Set("url", username)
	rawURL := apiBase + "/users/by_username?" + params.Encode()
	var wire wireProfile
	if err := c.getJSON(ctx, rawURL, &wire); err != nil {
		return User{}, err
	}
	return wireProfileToUser(wire), nil
}

// ─── HTTP internals ───────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	// check for DEV API error envelope
	var errEnv wireError
	if jsonErr := json.Unmarshal(body, &errEnv); jsonErr == nil && errEnv.Error != "" {
		if errEnv.Status == http.StatusNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("api error %d: %s", errEnv.Status, errEnv.Error)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func convertArticles(wire []wireArticle) []Article {
	out := make([]Article, len(wire))
	for i, w := range wire {
		out[i] = wireArticleToArticle(w)
	}
	return out
}

// ─── testable helpers (accept a base URL) ─────────────────────────────────────

func (c *Client) articlesWithBase(ctx context.Context, base string, top int, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 1000 {
		limit = 1000
	}
	params := url.Values{}
	params.Set("top", strconv.Itoa(top))
	params.Set("per_page", strconv.Itoa(limit))
	rawURL := base + "/articles?" + params.Encode()
	var wire []wireArticle
	if err := c.getJSON(ctx, rawURL, &wire); err != nil {
		return nil, err
	}
	return convertArticles(wire), nil
}

func (c *Client) tagArticlesWithBase(ctx context.Context, base string, tag string, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 1000 {
		limit = 1000
	}
	params := url.Values{}
	params.Set("tag", tag)
	params.Set("per_page", strconv.Itoa(limit))
	rawURL := base + "/articles?" + params.Encode()
	var wire []wireArticle
	if err := c.getJSON(ctx, rawURL, &wire); err != nil {
		return nil, err
	}
	return convertArticles(wire), nil
}

func (c *Client) userArticlesWithBase(ctx context.Context, base string, username string, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 1000 {
		limit = 1000
	}
	params := url.Values{}
	params.Set("username", username)
	params.Set("per_page", strconv.Itoa(limit))
	rawURL := base + "/articles?" + params.Encode()
	var wire []wireArticle
	if err := c.getJSON(ctx, rawURL, &wire); err != nil {
		return nil, err
	}
	return convertArticles(wire), nil
}

func (c *Client) articleByIDWithBase(ctx context.Context, base string, id int) (Article, error) {
	rawURL := fmt.Sprintf("%s/articles/%d", base, id)
	var wire wireArticle
	if err := c.getJSON(ctx, rawURL, &wire); err != nil {
		return Article{}, err
	}
	return wireArticleToArticle(wire), nil
}

func (c *Client) userWithBase(ctx context.Context, base string, username string) (User, error) {
	params := url.Values{}
	params.Set("url", username)
	rawURL := base + "/users/by_username?" + params.Encode()
	var wire wireProfile
	if err := c.getJSON(ctx, rawURL, &wire); err != nil {
		return User{}, err
	}
	return wireProfileToUser(wire), nil
}
