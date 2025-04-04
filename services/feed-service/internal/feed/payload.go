package feed

type CreateFeedRequest struct {
	UserID  string `json:"user_id"`
	PostID  string `json:"post_id"`
	Content string `json:"content"`
}

type CreateFeedResponse struct {
	Message string `json:"message"`
}
