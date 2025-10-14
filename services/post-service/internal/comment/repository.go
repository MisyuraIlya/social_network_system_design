package comment

import (
	"post-service/internal/shared/db"

	"gorm.io/gorm"
)

type Repository interface {
	Create(c *Comment) (*Comment, error)
	ListByPost(postID uint64, limit, offset int) ([]Comment, error)
	IncLike(commentID uint64) error
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(c *Comment) (*Comment, error) {
	if err := r.store.Base.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func (r *repo) ListByPost(postID uint64, limit, offset int) ([]Comment, error) {
	var out []Comment
	err := r.store.Base.Where("post_id = ?", postID).
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

func (r *repo) IncLike(commentID uint64) error {
	return r.store.Base.Model(&Comment{}).
		Where("id = ?", commentID).
		UpdateColumn("likes", gorm.Expr("likes + 1")).Error
}
