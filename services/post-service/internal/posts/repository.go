package posts

import (
	"fmt"

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

// NewRepository creates a new repository and auto-migrates the Post model.
func NewRepository(db *gorm.DB) Repository {
	if err := db.AutoMigrate(&Post{}); err != nil {
		panic("failed to auto-migrate Post model: " + err.Error())
	}
	return &repository{db: db}
}

func (r *repository) Create(post *Post) error {
	err := r.db.Create(post).Error
	if err != nil {
		fmt.Printf("Repository Create error: %v\n", err)
	} else {
		fmt.Printf("Post created with ID: %d\n", post.ID)
	}
	return err
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
