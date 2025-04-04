package message

import "gorm.io/gorm"

type Repository interface {
	Save(msg *Message) error
	FindAll() ([]Message, error)
}

type repository struct {
	db *gorm.DB
}

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
