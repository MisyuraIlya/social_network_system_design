package cache

import (
	"context"
	"feedback-service/configs"
	"log"

	"github.com/go-redis/redis/v8"
)

func NewRedis(cfg *configs.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr(),
	})

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Redis ping failed: %v", err)
	}

	return rdb
}
