package cities

import "gorm.io/gorm"

type Repository interface {
	Create(city *City) error
	List() ([]City, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(city *City) error {
	return r.db.Create(city).Error
}

func (r *repository) List() ([]City, error) {
	var cities []City
	err := r.db.Find(&cities).Error
	if err != nil {
		return nil, err
	}
	return cities, nil
}
