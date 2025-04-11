package feed

import "time"

type FeedItem struct {
	UserID    string    `json:"user_id"`
	PostID    string    `json:"post_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type PostMessage struct {
	ID          int       `json:"ID"`
	UserID      int       `json:"UserID"`
	Description string    `json:"Description"`
	Media       string    `json:"Media"`
	Likes       int       `json:"Likes"`
	Views       int       `json:"Views"`
	CreatedAt   time.Time `json:"CreatedAt"`
}
