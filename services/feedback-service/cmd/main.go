package main

import (
	"feedback-service/configs"
	"feedback-service/internal/feedback"
	"feedback-service/pkg/cache"
	"feedback-service/pkg/db"
	"log"
	"net/http"
)

func main() {
	cfg := configs.LoadConfig()
	dbInstance := db.NewDb(cfg)

	// 👇 Auto-create tables from models
	if err := dbInstance.DB.AutoMigrate(&feedback.Like{}, &feedback.Comment{}); err != nil {
		log.Fatalf("Failed to migrate DB: %v", err)
	}

	rdb := cache.NewRedis(cfg)

	router := http.NewServeMux()
	repo := feedback.NewRepository(dbInstance.DB, rdb)
	service := feedback.NewService(repo)

	feedback.NewHandler(router, feedback.HandlerDeps{
		Config:  cfg,
		Service: service,
	})

	log.Printf("Feedback service running on %s\n", cfg.AppPort)
	log.Fatal(http.ListenAndServe(cfg.AppPort, router))
}
