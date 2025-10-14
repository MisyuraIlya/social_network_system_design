package message

import (
	"message-service/internal/shared/db"
)

type Repository interface {
	Create(m *Message) (*Message, error)
	MarkSeen(messageID int64, userID string) error
	ListByChat(chatID int64, limit, offset int) ([]Message, error)
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(m *Message) (*Message, error) {
	if err := r.store.Base.Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (r *repo) MarkSeen(messageID int64, userID string) error {
	return r.store.Base.Model(&Message{}).Where("id=? AND user_id=?", messageID, userID).
		Update("is_seen", true).Error
}

func (r *repo) ListByChat(chatID int64, limit, offset int) ([]Message, error) {
	var out []Message
	err := r.store.Base.
		Where("chat_id = ?", chatID).
		Order("id DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}
