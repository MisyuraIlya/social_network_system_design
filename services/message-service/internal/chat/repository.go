package chat

import (
	"message-service/internal/shared/db"

	"gorm.io/gorm"
)

type Repository interface {
	Create(owner string, name string, extra []string) (*Chat, error)
	GetByID(chatID int64) (*Chat, error)
	AddUser(chatID int64, userID, typ string) error
	RemoveUser(chatID int64, userID string) error
	ListByUser(userID string, limit, offset int) ([]Chat, error)
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(owner, name string, extra []string) (*Chat, error) {
	c := &Chat{Name: name, OwnerID: owner}
	if err := r.store.Base.Create(c).Error; err != nil {
		return nil, err
	}
	members := append([]string{owner}, extra...)
	for _, m := range members {
		_ = r.store.Base.FirstOrCreate(&ChatUser{ChatID: c.ID, UserID: m, Type: "member"}).Error
	}
	return c, nil
}

func (r *repo) GetByID(chatID int64) (*Chat, error) {
	var c Chat
	if err := r.store.Base.First(&c, "id = ?", chatID).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repo) AddUser(chatID int64, userID, typ string) error {
	return r.store.Base.FirstOrCreate(&ChatUser{ChatID: chatID, UserID: userID, Type: typ}).Error
}

func (r *repo) RemoveUser(chatID int64, userID string) error {
	return r.store.Base.Delete(&ChatUser{}, "chat_id=? AND user_id=?", chatID, userID).Error
}

func (r *repo) ListByUser(userID string, limit, offset int) ([]Chat, error) {
	var out []Chat
	err := r.store.Base.
		Joins("JOIN chat_users cu ON cu.chat_id = chats.id AND cu.user_id = ?", userID).
		Order("chats.created_at DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

var _ = gorm.ErrRecordNotFound
