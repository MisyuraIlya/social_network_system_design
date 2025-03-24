package main

import (
	"fmt"
	"log"
	"net/http"

	"feedback-gateway/configs"
	"feedback-gateway/internal/feedback"
	"feedback-gateway/pkg/db"
)

func main() {
	cfg := configs.LoadConfig()
	database := db.NewDb(cfg)

	database.DB.AutoMigrate(&feedback.Like{}, &feedback.Comment{})

	repo := feedback.NewFeedbackRepository(database.DB)
	svc := feedback.NewFeedbackService(repo)
	handler := feedback.NewFeedbackHandler(svc)

	mux := http.NewServeMux()
	feedback.RegisterRoutes(mux, handler)

	srv := &http.Server{
		Addr:    cfg.AppPort,
		Handler: mux,
	}
	fmt.Printf("Feedback Service listening on %s\n", cfg.AppPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
