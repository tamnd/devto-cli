package devto

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(srv *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 0
	return NewClientWithConfig(cfg)
}

func sampleWireArticles(n int) []wireArticle {
	arts := make([]wireArticle, n)
	for i := range arts {
		arts[i] = wireArticle{
			ID:                   100 + i,
			Title:                "Article " + string(rune('A'+i)),
			URL:                  "https://dev.to/user/article-" + string(rune('a'+i)),
			PublicReactionsCount: 10 + i,
			CommentsCount:        i,
			TagList:              []string{"go", "programming"},
			User:                 wireUser{Username: "user" + string(rune('0'+i))},
		}
	}
	return arts
}

func TestTop(t *testing.T) {
	articles := sampleWireArticles(5)
	body, _ := json.Marshal(articles)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("top") != "1" {
			t.Errorf("expected top=1 query param")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	got, err := c.Top(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Errorf("got %d articles, want 5", len(got))
	}
	if got[0].Author != "user0" {
		t.Errorf("Author = %q, want user0", got[0].Author)
	}
}

func TestByTag(t *testing.T) {
	articles := sampleWireArticles(3)
	body, _ := json.Marshal(articles)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tag") != "go" {
			t.Errorf("expected tag=go query param, got %q", r.URL.Query().Get("tag"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	got, err := c.ByTag(context.Background(), "go", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("got %d articles, want 3", len(got))
	}
}

func TestByUser(t *testing.T) {
	articles := sampleWireArticles(2)
	body, _ := json.Marshal(articles)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("username") != "testuser" {
			t.Errorf("expected username=testuser, got %q", r.URL.Query().Get("username"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	got, err := c.ByUser(context.Background(), "testuser", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("got %d articles, want 2", len(got))
	}
}
