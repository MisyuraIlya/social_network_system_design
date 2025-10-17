package message

import "time"

type Message struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	UserID        string    `gorm:"size:64" json:"user_id"`
	ChatID        int64     `gorm:"index" json:"chat_id"`
	Text          string    `json:"text"`
	MediaURL      string    `gorm:"size:512" json:"media_url"`
	SendTime      time.Time `json:"send_time"`
	DeliveredTime time.Time `json:"delivered_time"`
}

type SendReq struct {
	ChatID   int64  `json:"chat_id" validate:"required"`
	Text     string `json:"text"`
	MediaURL string `json:"media_url"`
}

type MessageSeen struct {
	MessageID int64     `gorm:"primaryKey;index" json:"message_id"`
	UserID    string    `gorm:"primaryKey;size:64;index" json:"user_id"`
	SeenAt    time.Time `json:"seen_at"`
}
