package user

import "time"

type User struct {
	UserID   string `gorm:"uniqueIndex;size:64" json:"user_id"`
	ShardID  int    `gorm:"index" json:"shard_id"`
	ID       uint   `gorm:"primaryKey" json:"-"`
	Email    string `gorm:"uniqueIndex;size:100" json:"email"`
	Password string `gorm:"size:255" json:"-"`
	Name     string `gorm:"size:100" json:"name"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
