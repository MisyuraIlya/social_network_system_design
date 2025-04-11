package message

import "time"

// Message represents a chat message.
type Message struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id"`
	ChatID    uint      `json:"chat_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
