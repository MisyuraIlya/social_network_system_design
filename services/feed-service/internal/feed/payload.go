package feed

import "time"

type AddToFeedRequest struct {
	UserID uint `json:"userId"`
	PostID uint `json:"postId"`
}

type AddToFeedResponse struct {
	ID     uint `json:"id"`
	UserID uint `json:"userId"`
	PostID uint `json:"postId"`
}

type FeedItemData struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"userId"`
	PostID    uint      `json:"postId"`
	CreatedAt time.Time `json:"createdAt"`
}

type GetFeedResponse struct {
	Items []FeedItemData `json:"items"`
}
