package message

import "gorm.io/gorm"

// Repository defines database operations for messages.
type Repository interface {
	Save(msg *Message) error
	FindAll() ([]Message, error)
	UpdateMessage(id uint, newContent string) error
	DeleteMessage(id uint) error
	FindByChat(chatID uint) ([]Message, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new Repository and auto-migrates the Message model.
func NewRepository(db *gorm.DB) Repository {
	db.AutoMigrate(&Message{})
	return &repository{db: db}
}

func (r *repository) Save(msg *Message) error {
	return r.db.Create(msg).Error
}

func (r *repository) FindAll() ([]Message, error) {
	var messages []Message
	err := r.db.Order("created_at desc").Find(&messages).Error
	return messages, err
}

func (r *repository) UpdateMessage(id uint, newContent string) error {
	return r.db.Model(&Message{}).Where("id = ?", id).Update("content", newContent).Error
}

func (r *repository) DeleteMessage(id uint) error {
	return r.db.Delete(&Message{}, id).Error
}

func (r *repository) FindByChat(chatID uint) ([]Message, error) {
	var messages []Message
	err := r.db.Where("chat_id = ?", chatID).Order("created_at desc").Find(&messages).Error
	return messages, err
}
