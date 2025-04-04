package posts

import (
	"gorm.io/gorm"
)

type Repository interface {
	Create(post *Post) error
	GetAll() ([]Post, error)
	GetByID(id uint) (*Post, error)
	Update(post *Post) error
	Delete(id uint) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

func (r *repository) Create(post *Post) error {
	return r.db.Create(post).Error
}

func (r *repository) GetAll() ([]Post, error) {
	var posts []Post
	if err := r.db.Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *repository) GetByID(id uint) (*Post, error) {
	var post Post
	if err := r.db.First(&post, id).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *repository) Update(post *Post) error {
	return r.db.Save(post).Error
}

func (r *repository) Delete(id uint) error {
	return r.db.Delete(&Post{}, id).Error
}
