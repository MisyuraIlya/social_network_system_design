package comments

import "gorm.io/gorm"

type Repository interface {
	Create(c *Comment) error
	GetAllByPostID(postID uint) ([]Comment, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

func (r *repository) Create(c *Comment) error {
	return r.db.Create(c).Error
}

func (r *repository) GetAllByPostID(postID uint) ([]Comment, error) {
	var comments []Comment
	if err := r.db.Where("post_id = ?", postID).Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}
