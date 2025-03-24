package main

import (
	"fmt"
	"log"
	"net/http"

	"post-service/configs"
	"post-service/internal/post"
	"post-service/pkg/db"
)

func main() {
	cfg := configs.LoadConfig()
	database := db.NewDb(cfg)

	database.DB.AutoMigrate(&post.Post{})

	repo := post.NewPostRepository(database.DB)
	svc := post.NewPostService(repo)
	handler := post.NewPostHandler(svc)

	mux := http.NewServeMux()
	post.RegisterRoutes(mux, handler)

	srv := &http.Server{
		Addr:    cfg.AppPort,
		Handler: mux,
	}
	fmt.Printf("Post Service listening on %s\n", cfg.AppPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
