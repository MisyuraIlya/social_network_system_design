package message

import (
	"time"

	"message-service/pkg/redis"
)

type redisWrapper struct {
	client *redis.RedisClient
}

// NewRedisAdapter creates a new Cache adapter using Redis.
func NewRedisAdapter(r *redis.RedisClient) Cache {
	return &redisWrapper{client: r}
}

func (rw *redisWrapper) SetPopularChat(key string, value string) error {
	return rw.client.SetPopularChat(key, value, 24*time.Hour)
}

func (rw *redisWrapper) GetPopularChat(key string) (string, error) {
	return rw.client.GetPopularChat(key)
}

func (rw *redisWrapper) IncrChatPopularity(chatID uint, increment float64) error {
	return rw.client.IncrChatPopularity(chatID, increment)
}

func (rw *redisWrapper) GetTopPopularChats(limit int64) ([]string, error) {
	return rw.client.GetTopPopularChats(limit)
}
