package feed

import "time"

type FeedItem struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint `gorm:"index"`
	PostID uint `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
