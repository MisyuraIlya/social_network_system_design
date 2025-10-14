package like

import "time"

type PostLikesSum struct {
	PostID     uint64 `gorm:"primaryKey" json:"post_id"`
	LikesCount int64  `json:"likes_count"`
	UpdatedAt  time.Time
}

type PostLike struct {
	PostID    uint64 `gorm:"primaryKey;index" json:"post_id"`
	UserID    string `gorm:"primaryKey;size:64;index" json:"user_id"`
	CreatedAt time.Time
}
