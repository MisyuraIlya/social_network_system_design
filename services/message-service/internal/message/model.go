package message

import "time"

type Message struct {
	ID        uint   `gorm:"primaryKey"`
	DialogID  uint   `gorm:"index"`
	SenderID  uint   `gorm:"index"`
	Content   string `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
