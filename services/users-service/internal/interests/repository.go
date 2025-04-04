package interests

import "gorm.io/gorm"

type Repository interface {
	CreateInterest(i *Interest) error
	ListInterests() ([]Interest, error)
	AddUserInterest(iu *InterestUser) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateInterest(i *Interest) error {
	return r.db.Create(i).Error
}

func (r *repository) ListInterests() ([]Interest, error) {
	var result []Interest
	err := r.db.Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *repository) AddUserInterest(iu *InterestUser) error {
	return r.db.Create(iu).Error
}
