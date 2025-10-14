package comment

import "time"

type PostCommentsSum struct {
	PostID        uint64 `gorm:"primaryKey" json:"post_id"`
	CommentsCount int64  `json:"comments_count"`
	UpdatedAt     time.Time
}

type PostComment struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	PostID    uint64    `gorm:"index" json:"post_id"`
	UserID    string    `gorm:"size:64;index" json:"user_id"`
	ReplyID   *uint64   `json:"reply_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateReq struct {
	Text    string  `json:"text" validate:"required"`
	ReplyID *uint64 `json:"reply_id"`
}
