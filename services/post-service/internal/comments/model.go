package comments

import "time"

type Comment struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	PostID    uint
	Name      string
	Text      string
	CreatedAt time.Time
}
