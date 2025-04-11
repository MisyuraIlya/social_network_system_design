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

type UserFriends struct {
	UserID    uint      `json:"UserID"`
	FriendID  uint      `json:"FriendID"`
	CreatedAt time.Time `json:"CreatedAt"`
}
