package comment

import "time"

type Comment struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"index;size:64" json:"user_id"`
	PostID    uint64    `gorm:"index" json:"post_id"`
	Name      string    `gorm:"size:120" json:"name"`
	Text      string    `json:"text"`
	Likes     uint64    `json:"likes"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateReq struct {
	PostID uint64 `json:"post_id" validate:"required"`
	Text   string `json:"text" validate:"required"`
	Name   string `json:"name"`
}
