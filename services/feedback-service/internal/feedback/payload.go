package feedback

import "time"

type LikeRequest struct {
	UserID uint `json:"userId"`
	PostID uint `json:"postId"`
}

type LikeResponse struct {
	ID     uint `json:"id"`
	UserID uint `json:"userId"`
	PostID uint `json:"postId"`
}

type CommentRequest struct {
	UserID  uint   `json:"userId"`
	PostID  uint   `json:"postId"`
	Content string `json:"content"`
}

type CommentResponse struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"userId"`
	PostID    uint      `json:"postId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type GetCommentsResponse struct {
	Comments []CommentResponse `json:"comments"`
}

type CountLikesResponse struct {
	PostID    uint  `json:"postId"`
	LikeCount int64 `json:"likeCount"`
}
