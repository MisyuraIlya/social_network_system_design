package relationships

import "gorm.io/gorm"

type Repository interface {
	Create(r *Relationship) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(rel *Relationship) error {
	return r.db.Create(rel).Error
}
