package like

import (
	"context"
	"feedback-gateway/internal/shared/db"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Like(uid string, postID uint64) (int64, error)
	Unlike(uid string, postID uint64) (int64, error)
	GetCount(postID uint64, forUID string) (int64, bool, error)
}

type repo struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewRepository(s *db.Store, r *redis.Client) Repository {
	return &repo{db: s.DB, rdb: r}
}

func likeKey(postID uint64) string { return fmt.Sprintf("fb:likes:%d", postID) }

func (r *repo) Like(uid string, postID uint64) (int64, error) {
	ctx := context.Background()
	if err := r.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&PostLike{PostID: postID, UserID: uid}).Error; err != nil {
		return 0, err
	}
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "post_id"}},
		DoUpdates: clause.Assignments(map[string]any{"likes_count": gorm.Expr("post_likes_sums.likes_count + EXCLUDED.likes_count")}),
	}).Create(&PostLikesSum{PostID: postID, LikesCount: 1}).Error; err != nil {
		return 0, err
	}
	n, _ := r.rdb.Incr(ctx, likeKey(postID)).Result()
	if n <= 1 {
		var agg PostLikesSum
		if err := r.db.First(&agg, "post_id = ?", postID).Error; err == nil {
			_ = r.rdb.Set(ctx, likeKey(postID), agg.LikesCount, 0).Err()
			n = agg.LikesCount
		}
	}
	return n, nil
}

func (r *repo) Unlike(uid string, postID uint64) (int64, error) {
	ctx := context.Background()
	if err := r.db.Delete(&PostLike{}, "post_id=? AND user_id=?", postID, uid).Error; err != nil {
		return 0, err
	}
	if err := r.db.Exec(
		"UPDATE post_likes_sums SET likes_count = GREATEST(likes_count-1,0) WHERE post_id = ?",
		postID,
	).Error; err != nil {
		return 0, err
	}
	n, _ := r.rdb.Decr(ctx, likeKey(postID)).Result()
	if n < 0 {
		_ = r.rdb.Set(ctx, likeKey(postID), 0, 0).Err()
		n = 0
	}
	return n, nil
}

func (r *repo) GetCount(postID uint64, forUID string) (int64, bool, error) {
	ctx := context.Background()
	val, err := r.rdb.Get(ctx, likeKey(postID)).Int64()
	if err != nil {
		var agg PostLikesSum
		if e := r.db.First(&agg, "post_id = ?", postID).Error; e == nil {
			val = agg.LikesCount
			_ = r.rdb.Set(ctx, likeKey(postID), val, 0).Err()
		} else if e == gorm.ErrRecordNotFound {
			val = 0
		} else {
			return 0, false, e
		}
	}
	var exists int64
	if err := r.db.Model(&PostLike{}).
		Where("post_id = ? AND user_id = ?", postID, forUID).
		Count(&exists).Error; err != nil {
		return 0, false, err
	}
	return val, exists > 0, nil
}
