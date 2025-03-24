package feedback

import "time"

type Like struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"index"`
	PostID    uint `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Comment struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"index"`
	PostID    uint   `gorm:"index"`
	Content   string `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
