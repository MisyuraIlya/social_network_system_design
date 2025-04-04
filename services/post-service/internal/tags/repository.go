package tags

import (
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	CreateTag(tag *Tag) error
	GetAllTags() ([]Tag, error)
	CreatePostTag(postID, tagID uint) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

func (r *repository) CreateTag(tag *Tag) error {
	return r.db.Create(tag).Error
}

func (r *repository) GetAllTags() ([]Tag, error) {
	var tags []Tag
	if err := r.db.Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *repository) CreatePostTag(postID, tagID uint) error {
	pt := &PostTag{
		PostID:    postID,
		TagID:     tagID,
		CreatedAt: time.Now(),
	}
	return r.db.Create(pt).Error
}
