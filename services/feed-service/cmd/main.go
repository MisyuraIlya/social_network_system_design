package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"feed-service/configs"
	"feed-service/internal/feed"
	"feed-service/pkg/kafka"
	redisClient "feed-service/pkg/redis"

	kafkaGo "github.com/segmentio/kafka-go"
)

func main() {
	cfg := configs.LoadConfig()

	redisPool := redisClient.NewRedisPool(cfg)

	repo := feed.NewRepository(redisPool.Pool)
	svc := feed.NewService(repo, cfg.UserServiceURL)

	handler := feed.NewHandler(svc)
	router := handler.InitRoutes()

	consumer := kafka.NewConsumer(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		consumer.StartListening(ctx, func(m kafkaGo.Message) {
			err := svc.ConsumeNewPosts(ctx, m.Value)
			if err != nil {
				log.Printf("Failed to process post from Kafka: %v", err)
			} else {
				log.Printf("Processed message from Kafka, key=%s", string(m.Key))
			}
		})
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)
	go func() {
		<-exit
		log.Println("Shutting down Feed Service...")
		cancel()
		_ = consumer.Close()
		os.Exit(0)
	}()

	log.Printf("Feed Service is running on port %s", cfg.AppPort)
	srv := &http.Server{
		Addr:    cfg.AppPort,
		Handler: router,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
