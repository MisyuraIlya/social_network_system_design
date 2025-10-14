package post

import (
	"errors"

	"post-service/internal/shared/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(p *Post) (*Post, error)
	GetByID(id uint64) (*Post, error)
	ListByUser(userID string, limit, offset int) ([]Post, error)
	AttachTags(postID uint64, tagIDs []uint64) error
	IncLike(postID uint64) error
	IncView(postID uint64) error
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(p *Post) (*Post, error) {
	if err := r.store.Base.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (r *repo) GetByID(id uint64) (*Post, error) {
	var p Post
	if err := r.store.Base.First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repo) ListByUser(userID string, limit, offset int) ([]Post, error) {
	var out []Post
	err := r.store.Base.
		Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

func (r *repo) AttachTags(postID uint64, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	items := make([]PostTag, 0, len(tagIDs))
	for _, id := range tagIDs {
		items = append(items, PostTag{PostID: postID, TagID: id})
	}
	return r.store.Base.Clauses(clause.OnConflict{DoNothing: true}).Create(&items).Error
}

func (r *repo) IncLike(postID uint64) error {
	res := r.store.Base.Model(&Post{}).Where("id = ?", postID).UpdateColumn("likes", gorm.Expr("likes + 1"))
	return res.Error
}

func (r *repo) IncView(postID uint64) error {
	res := r.store.Base.Model(&Post{}).Where("id = ?", postID).UpdateColumn("views", gorm.Expr("views + 1"))
	return res.Error
}

var _ = errors.New
