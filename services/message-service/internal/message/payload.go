package message

import "time"

type SendMessageRequest struct {
	DialogID uint   `json:"dialogId"`
	SenderID uint   `json:"senderId"`
	Content  string `json:"content"`
}

type SendMessageResponse struct {
	ID        uint      `json:"id"`
	DialogID  uint      `json:"dialogId"`
	SenderID  uint      `json:"senderId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type MessageData struct {
	ID        uint      `json:"id"`
	DialogID  uint      `json:"dialogId"`
	SenderID  uint      `json:"senderId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type GetMessagesResponse struct {
	Messages []MessageData `json:"messages"`
}
