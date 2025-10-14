package migrate

import (
	"feedback-gateway/internal/comment"
	"feedback-gateway/internal/like"
	"feedback-gateway/internal/shared/db"
)

func AutoMigrateAll(store *db.Store) error {
	return store.DB.AutoMigrate(
		&like.PostLike{}, &like.PostLikesSum{},
		&comment.PostComment{}, &comment.PostCommentsSum{},
	)
}
