package main

import (
	"fmt"
	"log"
	"net/http"

	"message-service/configs"
	"message-service/internal/message"
	"message-service/pkg/db"
)

func main() {
	cfg := configs.LoadConfig()
	database := db.NewDb(cfg)

	database.DB.AutoMigrate(&message.Message{})

	repo := message.NewMessageRepository(database.DB)
	svc := message.NewMessageService(repo)
	handler := message.NewMessageHandler(svc)

	mux := http.NewServeMux()
	message.RegisterRoutes(mux, handler)

	srv := &http.Server{
		Addr:    cfg.AppPort,
		Handler: mux,
	}
	fmt.Printf("Message Service listening on %s\n", cfg.AppPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
