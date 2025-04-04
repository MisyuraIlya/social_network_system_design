package userdata

import "gorm.io/gorm"

type Repository interface {
	GetByUserID(userID int) (*UserData, error)
	CreateOrUpdate(data *UserData) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetByUserID(userID int) (*UserData, error) {
	var data UserData
	err := r.db.Where("user_id = ?", userID).First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *repository) CreateOrUpdate(data *UserData) error {
	return r.db.Save(data).Error
}
