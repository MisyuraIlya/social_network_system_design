package message

import "gorm.io/gorm"

type IMessageRepository interface {
	Create(msg *Message) (*Message, error)
	FindByDialogID(dialogID uint) ([]Message, error)
}

type MessageRepository struct {
	DB *gorm.DB
}

func NewMessageRepository(db *gorm.DB) IMessageRepository {
	return &MessageRepository{DB: db}
}

func (repo *MessageRepository) Create(msg *Message) (*Message, error) {
	if err := repo.DB.Create(msg).Error; err != nil {
		return nil, err
	}
	return msg, nil
}

func (repo *MessageRepository) FindByDialogID(dialogID uint) ([]Message, error) {
	var messages []Message
	if err := repo.DB.Where("dialog_id = ?", dialogID).
		Order("created_at asc").
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}
