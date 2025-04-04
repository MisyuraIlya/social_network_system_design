package likes

import "gorm.io/gorm"

type Repository interface {
	AddLike(l *Like) error
	IncrementPostLikes(postID uint) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

func (r *repository) AddLike(l *Like) error {
	return r.db.Create(l).Error
}

func (r *repository) IncrementPostLikes(postID uint) error {
	// Use a raw query or GORM's updates:
	return r.db.Exec(`UPDATE posts SET likes = likes + 1 WHERE id = ?`, postID).Error
}
