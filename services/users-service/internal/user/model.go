package user

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Email     string `gorm:"uniqueIndex;size:100"`
	Password  string `gorm:"size:255"`
	Name      string `gorm:"size:100"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
