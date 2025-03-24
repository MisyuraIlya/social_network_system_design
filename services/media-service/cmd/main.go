package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"media-service/configs"
	"media-service/internal/media"
	"media-service/pkg/db"
)

func main() {
	cfg := configs.LoadConfig()

	if _, err := os.Stat(cfg.StorageDir); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(cfg.StorageDir, 0755); mkErr != nil {
			log.Fatalf("Unable to create storage directory: %v", mkErr)
		}
	}

	database := db.NewDb(cfg)
	database.DB.AutoMigrate(&media.Media{})

	repo := media.NewMediaRepository(database.DB)
	svc := media.NewMediaService(repo, cfg.StorageDir)
	handler := media.NewMediaHandler(svc, cfg.StorageDir)

	mux := http.NewServeMux()
	media.RegisterRoutes(mux, handler)

	srv := &http.Server{
		Addr:    cfg.AppPort,
		Handler: mux,
	}
	fmt.Printf("Media Service listening on %s\n", cfg.AppPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
