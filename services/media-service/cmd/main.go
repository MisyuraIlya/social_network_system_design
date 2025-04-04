package main

import (
	"log"
	"net/http"

	"media-service/configs"
	"media-service/internal/media"
)

func main() {
	cfg, err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	handler := media.InitializeHandler(cfg)

	srv := &http.Server{
		Addr:    cfg.AppPort,
		Handler: handler.InitRoutes(),
	}

	log.Printf("Starting Media Service on port %s\n", cfg.AppPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
