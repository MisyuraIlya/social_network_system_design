package posts

import (
	"time"
)

type Post struct {
	ID          uint `gorm:"primaryKey"`
	UserID      uint
	Description string
	Media       string
	Likes       int
	Views       int
	CreatedAt   time.Time
}
