package posts

import "time"

type CreatePostRequest struct {
	UserID  uint   `json:"userId"`
	Content string `json:"content"`
}

type CreatePostResponse struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"userId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type GetPostResponse struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"userId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ListPostsResponse struct {
	Posts []GetPostResponse `json:"posts"`
}
