package feed

import "time"

type PostEvent struct {
	ID          int64     `json:"id"`
	UserID      string    `json:"user_id"`
	Description string    `json:"description"`
	MediaURL    string    `json:"media_url"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	Likes       int64     `json:"likes,omitempty"`
	Views       int64     `json:"views,omitempty"`
}

type FeedEntry struct {
	PostID    int64     `json:"post_id"`
	AuthorID  string    `json:"author_id"`
	MediaURL  string    `json:"media_url,omitempty"`
	Snippet   string    `json:"snippet,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Score     float64   `json:"score"`
}
