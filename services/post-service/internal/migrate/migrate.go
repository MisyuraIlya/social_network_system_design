package migrate

import (
	"post-service/internal/post"
	"post-service/internal/shared/db"
	"post-service/internal/tag"
)

func AutoMigrateAll(store *db.Store) error {
	return store.Base.AutoMigrate(
		&post.Post{},
		&post.PostTag{},
		&tag.Tag{},
	)
}
