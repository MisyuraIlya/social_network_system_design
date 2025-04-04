package feedback

type LikeRequest struct {
	UserID string `json:"userId"`
	PostID string `json:"postId"`
}

type CommentRequest struct {
	UserID  string `json:"userId"`
	PostID  string `json:"postId"`
	Content string `json:"content"`
}
