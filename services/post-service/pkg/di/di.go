package di

import (
	"post-service/configs"
	"post-service/internal/comments"
	"post-service/internal/likes"
	"post-service/internal/posts"
	"post-service/internal/tags"
	"post-service/pkg/db"
	"post-service/pkg/kafka"

	"gorm.io/gorm"
)

type Container struct {
	DB             *gorm.DB
	PostService    posts.Service
	CommentService comments.Service
	LikeService    likes.Service
	TagService     tags.Service
	KafkaProducer  *kafka.Producer
}

func BuildContainer(cfg *configs.Config) *Container {
	dbConn := db.NewDb(cfg)

	kafkaProducer := kafka.NewProducer(cfg.KafkaBrokerURL, cfg.KafkaTopic)

	postRepo := posts.NewRepository(dbConn.DB)
	postService := posts.NewService(postRepo, kafkaProducer, cfg)

	commentRepo := comments.NewRepository(dbConn.DB)
	commentService := comments.NewService(commentRepo)

	likeRepo := likes.NewRepository(dbConn.DB)
	likeService := likes.NewService(likeRepo)

	tagRepo := tags.NewRepository(dbConn.DB)
	tagService := tags.NewService(tagRepo)

	return &Container{
		DB:             dbConn.DB,
		PostService:    postService,
		CommentService: commentService,
		LikeService:    likeService,
		TagService:     tagService,
		KafkaProducer:  kafkaProducer,
	}
}
