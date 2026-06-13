package devto

import (
	"fmt"
	"strings"
	"time"
)

// Article is the record emitted for DEV articles.
type Article struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	By          string `json:"by"`
	Reactions   int    `json:"reactions"`
	Comments    int    `json:"comments"`
	ReadingTime int    `json:"reading_time"`
	Tags        string `json:"tags"`
	Published   string `json:"published"`
}

// User is the record emitted for DEV user profiles.
type User struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Location string `json:"location"`
	Joined   string `json:"joined"`
	Posts    int    `json:"posts"`
	URL      string `json:"url"`
}

// ─── wire types from DEV API ──────────────────────────────────────────────────

type wireUser struct {
	Username string `json:"username"`
	Name     string `json:"name"`
}

type wireArticle struct {
	ID                   int      `json:"id"`
	Title                string   `json:"title"`
	Description          string   `json:"description"`
	URL                  string   `json:"url"`
	PublishedAt          string   `json:"published_at"`
	TagList              []string `json:"tag_list"`
	User                 wireUser `json:"user"`
	PublicReactionsCount int      `json:"public_reactions_count"`
	CommentsCount        int      `json:"comments_count"`
	ReadingTimeMinutes   int      `json:"reading_time_minutes"`
	CoverImage           *string  `json:"cover_image"`
}

type wireProfile struct {
	Username      string `json:"username"`
	Name          string `json:"name"`
	Summary       string `json:"summary"`
	Location      string `json:"location"`
	JoinedAt      string `json:"joined_at"`
	ArticlesCount int    `json:"articles_count"`
}

type wireError struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// ─── converters ──────────────────────────────────────────────────────────────

func wireArticleToArticle(w wireArticle) Article {
	published := w.PublishedAt
	if published == "" {
		published = ""
	} else {
		// normalize to RFC3339 if possible
		if t, err := time.Parse(time.RFC3339, published); err == nil {
			published = t.UTC().Format(time.RFC3339)
		}
	}
	tags := strings.Join(w.TagList, ";")
	return Article{
		ID:          w.ID,
		Title:       w.Title,
		Description: w.Description,
		URL:         w.URL,
		By:          w.User.Username,
		Reactions:   w.PublicReactionsCount,
		Comments:    w.CommentsCount,
		ReadingTime: w.ReadingTimeMinutes,
		Tags:        tags,
		Published:   published,
	}
}

func wireProfileToUser(p wireProfile) User {
	joined := p.JoinedAt
	if joined == "" {
		joined = ""
	} else {
		// DEV returns "Jan  1, 2024" or ISO; normalize what we can
		if t, err := time.Parse(time.RFC3339, joined); err == nil {
			joined = t.UTC().Format(time.RFC3339)
		}
	}
	return User{
		Username: p.Username,
		Name:     p.Name,
		Summary:  p.Summary,
		Location: p.Location,
		Joined:   joined,
		Posts:    p.ArticlesCount,
		URL:      fmt.Sprintf("https://dev.to/%s", p.Username),
	}
}
