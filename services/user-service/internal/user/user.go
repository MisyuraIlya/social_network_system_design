package user

import "time"

type User struct {
	UserID    string    `gorm:"uniqueIndex;size:64" json:"user_id"`
	ShardID   int       `gorm:"index" json:"shard_id"`
	ID        uint      `gorm:"primaryKey" json:"-"`
	Email     string    `gorm:"uniqueIndex;size:120" json:"email"`
	PassHash  string    `gorm:"size:255" json:"-"`
	Name      string    `gorm:"size:100" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RegisterReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required"`
}
type LoginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}
