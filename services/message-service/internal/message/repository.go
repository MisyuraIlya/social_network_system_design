package message

import (
	"time"

	"message-service/internal/shared/db"

	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(m *Message) (*Message, error)
	MarkSeen(messageID int64, userID string) error
	ListByChat(chatID int64, limit, offset int) ([]Message, error)

	// NEW:
	GetByID(messageID int64) (*Message, error)
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
	ms := &MessageSeen{
		MessageID: messageID,
		UserID:    userID,
		SeenAt:    time.Now(),
	}
	return r.store.Base.Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "message_id"}, {Name: "user_id"}},
			DoNothing: true,
		},
	).Create(ms).Error
}

func (r *repo) ListByChat(chatID int64, limit, offset int) ([]Message, error) {
	var out []Message
	err := r.store.Base.
		Where("chat_id = ?", chatID).
		Order("id DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

func (r *repo) GetByID(messageID int64) (*Message, error) {
	var m Message
	if err := r.store.Base.First(&m, "id = ?", messageID).Error; err != nil {
		return nil, err
	}
	return &m, nil
}
