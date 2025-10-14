package comment

import (
	"context"
	"feedback-gateway/internal/shared/db"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(uid string, postID uint64, in CreateReq) (*PostComment, error)
	DeleteMine(uid string, commentID uint64) error
	ListByPost(postID uint64, limit, offset int) ([]PostComment, error)
	Counts(postID uint64) (likes int64, comments int64, err error)
	IncSum(postID uint64, delta int) error
}

type repo struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewRepository(s *db.Store, r *redis.Client) Repository {
	return &repo{db: s.DB, rdb: r}
}

func ckey(postID uint64) string { return fmt.Sprintf("fb:comments:%d", postID) }

func (r *repo) Create(uid string, postID uint64, in CreateReq) (*PostComment, error) {
	pc := &PostComment{PostID: postID, UserID: uid, ReplyID: in.ReplyID, Text: in.Text}
	if err := r.db.Create(pc).Error; err != nil {
		return nil, err
	}
	_ = r.IncSum(postID, +1)
	return pc, nil
}

func (r *repo) DeleteMine(uid string, commentID uint64) error {
	var c PostComment
	if err := r.db.First(&c, "id = ? AND user_id = ?", commentID, uid).Error; err != nil {
		return err
	}
	if err := r.db.Delete(&PostComment{}, "id = ?", commentID).Error; err != nil {
		return err
	}
	return r.IncSum(c.PostID, -1)
}

func (r *repo) IncSum(postID uint64, delta int) error {
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "post_id"}},
		DoUpdates: clause.Assignments(map[string]any{"comments_count": gorm.Expr("post_comments_sums.comments_count + EXCLUDED.comments_count")}),
	}).Create(&PostCommentsSum{PostID: postID, CommentsCount: int64(delta)}).Error; err != nil {
		return err
	}
	ctx := context.Background()
	if delta > 0 {
		_, _ = r.rdb.Incr(ctx, ckey(postID)).Result()
	} else {
		_, _ = r.rdb.Decr(ctx, ckey(postID)).Result()
	}
	return nil
}

func (r *repo) ListByPost(postID uint64, limit, offset int) ([]PostComment, error) {
	var out []PostComment
	err := r.db.Where("post_id = ?", postID).
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

func (r *repo) Counts(postID uint64) (int64, int64, error) {
	var cs PostCommentsSum
	var comments int64
	if err := r.db.First(&cs, "post_id = ?", postID).Error; err == nil {
		comments = cs.CommentsCount
	} else if err == gorm.ErrRecordNotFound {
		comments = 0
	} else {
		return 0, 0, err
	}
	return 0, comments, nil
}
