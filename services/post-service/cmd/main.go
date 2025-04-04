package main

import (
	"fmt"
	"net/http"
	"post-service/configs"
	"post-service/internal/comments"
	"post-service/internal/likes"
	"post-service/internal/posts"
	"post-service/internal/tags"
	"post-service/pkg/di"
)

func App() http.Handler {
	cfg := configs.LoadConfig()
	container := di.BuildContainer(cfg)

	container.DB.AutoMigrate(
		&posts.Post{},
		&comments.Comment{},
		&likes.Like{},
		&tags.Tag{},
		&tags.PostTag{},
	)

	router := http.NewServeMux()

	// Register post routes
	posts.NewHandler(router, posts.HandlerDeps{
		Config:  cfg,
		Service: container.PostService,
	})

	// Register comments
	comments.NewHandler(router, comments.HandlerDeps{
		Config:  cfg,
		Service: container.CommentService,
	})

	// Register likes
	likes.NewHandler(router, likes.HandlerDeps{
		Config:  cfg,
		Service: container.LikeService,
	})

	// Register tags
	tags.NewHandler(router, tags.HandlerDeps{
		Config:  cfg,
		Service: container.TagService,
	})

	return router
}

func main() {
	app := App()
	server := http.Server{
		Addr:    ":8082",
		Handler: app,
	}
	fmt.Println("Post Service listening on port 8082")
	server.ListenAndServe()
}
