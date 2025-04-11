package main

import (
	"log"
	"net/http"
	"strings"

	"message-service/configs"
	"message-service/internal/message"
	"message-service/pkg/db"
	kafkapkg "message-service/pkg/kafka"
	redispkg "message-service/pkg/redis"
)

func main() {
	cfg := configs.LoadConfig()

	pg := db.NewDb(cfg)
	repo := message.NewRepository(pg.DB)

	// Redis client
	rds := redispkg.NewRedisClient(cfg.RedisHost, cfg.RedisPort)

	// Kafka producer
	kafkaProducer := kafkapkg.NewProducer(strings.Split(cfg.KafkaBrokers, ","), cfg.KafkaTopic)
	redisAdapter := message.NewRedisAdapter(rds)

	// Init dependencies
	cache := message.NewCache(redisAdapter)
	publisher := message.NewPublisher(kafkaProducer)
	// Pass publisher and mediaSvcURL to NewService
	service := message.NewService(repo, cache, publisher, cfg.MediaServiceURL)

	// Router setup
	router := http.NewServeMux()
	message.NewHandler(router, message.HandlerDeps{
		Config:    cfg,
		Service:   service,
		Cache:     cache,
		Publisher: publisher,
	})

	log.Println("Message service running on", cfg.AppPort)
	log.Fatal(http.ListenAndServe(cfg.AppPort, router))
}
