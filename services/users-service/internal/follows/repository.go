package follows

import "gorm.io/gorm"

type Repository interface {
	CreateFollow(f *Follow) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateFollow(f *Follow) error {
	return r.db.Create(f).Error
}
