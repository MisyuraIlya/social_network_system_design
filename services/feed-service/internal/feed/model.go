package feed

import "time"

// FeedItem is a data structure representing a post in the feed.
type FeedItem struct {
	UserID    string    `json:"user_id"`
	PostID    string    `json:"post_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
