package media

import "gorm.io/gorm"

type IMediaRepository interface {
	Create(m *Media) (*Media, error)
	FindByID(id uint) (*Media, error)
}

type MediaRepository struct {
	DB *gorm.DB
}

func NewMediaRepository(db *gorm.DB) IMediaRepository {
	return &MediaRepository{DB: db}
}

func (repo *MediaRepository) Create(m *Media) (*Media, error) {
	if err := repo.DB.Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (repo *MediaRepository) FindByID(id uint) (*Media, error) {
	var media Media
	if err := repo.DB.First(&media, id).Error; err != nil {
		return nil, err
	}
	return &media, nil
}
