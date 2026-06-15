package devto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const Host = "dev.to"
const baseURL = "https://dev.to/api"
const DefaultUserAgent = "Mozilla/5.0 (compatible; devto-cli/0.1; +https://github.com/tamnd/devto-cli)"

type Config struct {
	BaseURL   string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
	UserAgent string
}

func DefaultConfig() Config {
	return Config{
		BaseURL:   baseURL,
		Rate:      time.Second,
		Retries:   3,
		Timeout:   30 * time.Second,
		UserAgent: DefaultUserAgent,
	}
}

type Client struct {
	cfg  Config
	http *http.Client
	last time.Time
}

func NewClient() *Client { return NewClientWithConfig(DefaultConfig()) }

func NewClientWithConfig(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
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
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// wireUser is the nested user object in article responses.
type wireUser struct {
	Username string `json:"username"`
}

// wireArticle is the raw API response for a DEV.to article.
type wireArticle struct {
	ID                   int      `json:"id"`
	Title                string   `json:"title"`
	URL                  string   `json:"url"`
	PublicReactionsCount int      `json:"public_reactions_count"`
	CommentsCount        int      `json:"comments_count"`
	TagList              []string `json:"tag_list"`
	User                 wireUser `json:"user"`
}

// Article is one DEV.to article.
type Article struct {
	ID        string   `json:"id"        kit:"id" table:"id"`
	Title     string   `json:"title"              table:"title"`
	Author    string   `json:"author"             table:"author"`
	Reactions int      `json:"reactions"          table:"reactions"`
	Comments  int      `json:"comments"           table:"comments"`
	Tags      []string `json:"tags"               table:"tags"`
	URL       string   `json:"url"                table:"url,url"`
}

func fromWire(w wireArticle) *Article {
	return &Article{
		ID:        fmt.Sprintf("%d", w.ID),
		Title:     w.Title,
		Author:    w.User.Username,
		Reactions: w.PublicReactionsCount,
		Comments:  w.CommentsCount,
		Tags:      w.TagList,
		URL:       w.URL,
	}
}

func (c *Client) fetchArticles(ctx context.Context, endpoint string, limit int) ([]*Article, error) {
	base := c.cfg.BaseURL
	if base == "" {
		base = baseURL
	}
	body, err := c.get(ctx, base+endpoint)
	if err != nil {
		return nil, err
	}
	var raw []wireArticle
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	articles := make([]*Article, 0, len(raw))
	for _, w := range raw {
		articles = append(articles, fromWire(w))
	}
	if limit > 0 && len(articles) > limit {
		articles = articles[:limit]
	}
	return articles, nil
}

// Top returns the top DEV.to articles.
func (c *Client) Top(ctx context.Context, limit int) ([]*Article, error) {
	q := url.Values{}
	q.Set("top", "1")
	if limit > 0 {
		q.Set("per_page", fmt.Sprintf("%d", limit))
	} else {
		q.Set("per_page", "20")
	}
	return c.fetchArticles(ctx, "/articles?"+q.Encode(), 0)
}

// ByTag returns DEV.to articles for a tag.
func (c *Client) ByTag(ctx context.Context, tag string, limit int) ([]*Article, error) {
	q := url.Values{}
	q.Set("tag", tag)
	if limit > 0 {
		q.Set("per_page", fmt.Sprintf("%d", limit))
	} else {
		q.Set("per_page", "20")
	}
	return c.fetchArticles(ctx, "/articles?"+q.Encode(), 0)
}

// ByUser returns DEV.to articles by a user.
func (c *Client) ByUser(ctx context.Context, username string, limit int) ([]*Article, error) {
	q := url.Values{}
	q.Set("username", username)
	if limit > 0 {
		q.Set("per_page", fmt.Sprintf("%d", limit))
	} else {
		q.Set("per_page", "20")
	}
	return c.fetchArticles(ctx, "/articles?"+q.Encode(), 0)
}
