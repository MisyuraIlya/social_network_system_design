package redis

import (
	"context"
	"fmt"
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
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: "",
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

func (r *RedisClient) IncrChatPopularity(chatID uint, increment float64) error {
	return r.Client.ZIncrBy(r.Ctx, "popular_chats", increment, fmt.Sprintf("%d", chatID)).Err()
}

func (r *RedisClient) GetTopPopularChats(limit int64) ([]string, error) {
	return r.Client.ZRevRange(r.Ctx, "popular_chats", 0, limit-1).Result()
}
