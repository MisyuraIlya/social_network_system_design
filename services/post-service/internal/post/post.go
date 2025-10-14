package post

import "time"

type Post struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"index;size:64" json:"user_id"`
	Description string    `json:"description"`
	MediaURL    string    `gorm:"size:512" json:"media"`
	Likes       uint64    `json:"likes"`
	Views       uint64    `json:"views"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PostTag struct {
	PostID uint64 `gorm:"primaryKey"`
	TagID  uint64 `gorm:"primaryKey"`
}

type CreateReq struct {
	Description string   `json:"description" validate:"required"`
	MediaURL    string   `json:"media_url"`
	Tags        []string `json:"tags"`
}

type LikeReq struct {
}
