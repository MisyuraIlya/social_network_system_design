// pkg/di/di.go
package di

import (
	"gorm.io/gorm"

	"post-service/configs"
	"post-service/internal/comments"
	"post-service/internal/likes"
	"post-service/internal/posts"
	"post-service/internal/tags"
	"post-service/pkg/db"
)

type Container struct {
	DB             *gorm.DB
	PostService    posts.Service
	CommentService comments.Service
	LikeService    likes.Service
	TagService     tags.Service
}

func BuildContainer(cfg *configs.Config) *Container {
	// 1) Open DB connection
	dbConn := db.NewDb(cfg)

	// 2) Build repositories & services
	postRepo := posts.NewRepository(dbConn.DB)
	postService := posts.NewService(postRepo)

	commentRepo := comments.NewRepository(dbConn.DB)
	commentService := comments.NewService(commentRepo)

	likeRepo := likes.NewRepository(dbConn.DB)
	likeService := likes.NewService(likeRepo)

	tagRepo := tags.NewRepository(dbConn.DB)
	tagService := tags.NewService(tagRepo)

	// 3) Return container
	return &Container{
		DB:             dbConn.DB,
		PostService:    postService,
		CommentService: commentService,
		LikeService:    likeService,
		TagService:     tagService,
	}
}
