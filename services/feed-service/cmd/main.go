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

	svc := feed.NewService(repo, cfg.UserServiceURL, cfg.PostServiceURL)

	router := http.NewServeMux()
	feed.NewHandler(router, svc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := kafka.NewConsumer(cfg)

	go func() {
		consumer.StartListening(ctx, func(m kafkaGo.Message) {
			err := svc.ConsumeNewPosts(ctx, m.Value)
			if err != nil {
				log.Printf("Failed to process Kafka message: %v", err)
			} else {
				log.Printf("Successfully processed Kafka message with key: %s", string(m.Key))
			}
		})
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)

	go func() {
		<-exit
		log.Println("Shutting down Feed Service gracefully...")
		cancel()
		if err := consumer.Close(); err != nil {
			log.Printf("Error closing Kafka consumer: %v", err)
		}
		os.Exit(0)
	}()

	log.Printf("Feed Service is running on port %s", cfg.AppPort)
	server := &http.Server{
		Addr:    cfg.AppPort,
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}
