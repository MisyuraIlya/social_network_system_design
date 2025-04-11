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
	posts.NewHandler(router, cfg, container.PostService)
	comments.NewHandler(router, cfg, container.CommentService)
	likes.NewHandler(router, cfg, container.LikeService)
	tags.NewHandler(router, cfg, container.TagService)
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
