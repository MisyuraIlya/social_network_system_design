package tags

import "time"

type Tag struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	CreatedAt time.Time
}

type PostTag struct {
	PostID    uint
	TagID     uint
	CreatedAt time.Time
}
