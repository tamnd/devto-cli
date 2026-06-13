package devto

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient returns a Client pointed at srv with pacing disabled.
func newTestClient(srv *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 2
	c := NewClient(cfg)
	// override the base URL via a round-tripper that rewrites the host
	c.httpClient = srv.Client()
	return c
}

// makeArticleJSON returns a JSON array with one article.
func makeArticleJSON(id int, title, tag, username string) []byte {
	a := wireArticle{
		ID:                   id,
		Title:                title,
		Description:          "A description.",
		URL:                  "https://dev.to/user/article",
		PublishedAt:          "2024-01-15T10:00:00Z",
		TagList:              []string{tag},
		User:                 wireUser{Username: username, Name: "Test User"},
		PublicReactionsCount: 42,
		CommentsCount:        7,
		ReadingTimeMinutes:   5,
	}
	b, _ := json.Marshal([]wireArticle{a})
	return b
}

func makeSingleArticleJSON(id int, title string) []byte {
	a := wireArticle{
		ID:                   id,
		Title:                title,
		Description:          "Single article.",
		URL:                  "https://dev.to/user/single",
		PublishedAt:          "2024-02-20T12:00:00Z",
		TagList:              []string{"go"},
		User:                 wireUser{Username: "alice", Name: "Alice"},
		PublicReactionsCount: 100,
		CommentsCount:        20,
		ReadingTimeMinutes:   8,
	}
	b, _ := json.Marshal(a)
	return b
}

func makeUserJSON(username string) []byte {
	p := wireProfile{
		Username:      username,
		Name:          "Ben Halpern",
		Summary:       "Founder of DEV.",
		Location:      "New York",
		JoinedAt:      "2015-06-01T00:00:00Z",
		ArticlesCount: 512,
	}
	b, _ := json.Marshal(p)
	return b
}

func TestArticles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/articles" {
			http.NotFound(w, r)
			return
		}
		top := r.URL.Query().Get("top")
		if top != "7" {
			t.Errorf("expected top=7, got %q", top)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeArticleJSON(1, "Top Story", "go", "alice"))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)
	articles, err := c.articlesWithBase(context.Background(), srv.URL+"/api", 7, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 1 {
		t.Fatalf("expected 1 article, got %d", len(articles))
	}
	if articles[0].Title != "Top Story" {
		t.Errorf("title = %q, want %q", articles[0].Title, "Top Story")
	}
	if articles[0].By != "alice" {
		t.Errorf("by = %q, want %q", articles[0].By, "alice")
	}
	if articles[0].Tags != "go" {
		t.Errorf("tags = %q, want %q", articles[0].Tags, "go")
	}
}

func TestTagArticles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/articles" {
			http.NotFound(w, r)
			return
		}
		tag := r.URL.Query().Get("tag")
		if tag != "python" {
			t.Errorf("expected tag=python, got %q", tag)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeArticleJSON(2, "Python Post", "python", "bob"))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)
	articles, err := c.tagArticlesWithBase(context.Background(), srv.URL+"/api", "python", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 1 || articles[0].By != "bob" {
		t.Errorf("unexpected articles: %+v", articles)
	}
}

func TestUserArticles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/articles" {
			http.NotFound(w, r)
			return
		}
		u := r.URL.Query().Get("username")
		if u != "carol" {
			t.Errorf("expected username=carol, got %q", u)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeArticleJSON(3, "Carol's Post", "webdev", "carol"))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)
	articles, err := c.userArticlesWithBase(context.Background(), srv.URL+"/api", "carol", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 1 || articles[0].By != "carol" {
		t.Errorf("unexpected articles: %+v", articles)
	}
}

func TestArticleByID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/articles/99" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(makeSingleArticleJSON(99, "Single Article"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)

	// articleByIDWithBase uses base directly (no /api prefix needed for single article)
	a, err := c.articleByIDWithBase(context.Background(), srv.URL, 99)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != 99 {
		t.Errorf("id = %d, want 99", a.ID)
	}
}

func TestArticleByIDNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)

	_, err := c.articleByIDWithBase(context.Background(), srv.URL, 9999)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/by_username" {
			http.NotFound(w, r)
			return
		}
		u := r.URL.Query().Get("url")
		if u != "ben" {
			t.Errorf("expected url=ben, got %q", u)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeUserJSON("ben"))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)

	user, err := c.userWithBase(context.Background(), srv.URL+"/api", "ben")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "ben" {
		t.Errorf("username = %q, want %q", user.Username, "ben")
	}
	if user.Posts != 512 {
		t.Errorf("posts = %d, want 512", user.Posts)
	}
}

func TestUserNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := NewClient(cfg)

	_, err := c.userWithBase(context.Background(), srv.URL+"/api", "nobody")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRetryOn429(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(makeArticleJSON(5, "Retry Test", "go", "dave"))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 3
	c := NewClient(cfg)

	articles, err := c.articlesWithBase(context.Background(), srv.URL+"/api", 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 1 {
		t.Errorf("expected 1 article after retry, got %d", len(articles))
	}
	if hits < 2 {
		t.Errorf("expected at least 2 server hits for retry, got %d", hits)
	}
}
