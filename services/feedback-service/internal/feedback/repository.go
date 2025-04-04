package feedback

import (
	"context"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type Repository struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewRepository(db *gorm.DB, redis *redis.Client) *Repository {
	return &Repository{db: db, redis: redis}
}

func (r *Repository) SaveLike(like Like) error {
	if err := r.db.Create(&like).Error; err != nil {
		return err
	}
	return r.redis.Incr(context.Background(), "likes:"+like.PostID).Err()
}

func (r *Repository) SaveComment(comment Comment) error {
	if err := r.db.Create(&comment).Error; err != nil {
		return err
	}
	return r.redis.Incr(context.Background(), "comments:"+comment.PostID).Err()
}
