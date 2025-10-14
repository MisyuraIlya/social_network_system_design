package chat

import "time"

type Chat struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:200" json:"name"`
	OwnerID   string    `gorm:"size:64" json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatUser struct {
	ChatID    int64     `gorm:"primaryKey" json:"chat_id"`
	UserID    string    `gorm:"primaryKey;size:64" json:"user_id"`
	Type      string    `gorm:"size:32" json:"type"` // member/admin/…
	CreatedAt time.Time `json:"created_at"`
}

type CreateReq struct {
	Name    string   `json:"name" validate:"required"`
	Members []string `json:"members"` // optional (besides creator)
}
