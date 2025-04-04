package redis

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
	Ctx    context.Context
}

func NewRedisClient(host, port string) *RedisClient {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: "", // no password
		DB:       0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	return &RedisClient{Client: rdb, Ctx: ctx}
}

func (r *RedisClient) SetPopularChat(key string, value string, ttl time.Duration) error {
	return r.Client.Set(r.Ctx, key, value, ttl).Err()
}

func (r *RedisClient) GetPopularChat(key string) (string, error) {
	return r.Client.Get(r.Ctx, key).Result()
}
