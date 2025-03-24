package main

import (
	"fmt"
	"log"
	"net/http"

	"feed-service/configs"
	"feed-service/internal/feed"
	"feed-service/pkg/db"
)

func main() {
	cfg := configs.LoadConfig()
	database := db.NewDb(cfg)

	database.DB.AutoMigrate(&feed.FeedItem{})

	repo := feed.NewFeedRepository(database.DB)
	svc := feed.NewFeedService(repo)
	handler := feed.NewFeedHandler(svc)

	mux := http.NewServeMux()
	feed.RegisterRoutes(mux, handler)

	srv := &http.Server{
		Addr:    cfg.AppPort,
		Handler: mux,
	}
	fmt.Printf("Feed Service listening on %s\n", cfg.AppPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
