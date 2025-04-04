package friends

import "gorm.io/gorm"

type Repository interface {
	CreateFriend(f *Friend) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateFriend(f *Friend) error {
	return r.db.Create(f).Error
}
