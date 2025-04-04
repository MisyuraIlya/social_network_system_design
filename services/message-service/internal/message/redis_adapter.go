package message

import (
	"message-service/pkg/redis"
	"time"
)

type redisWrapper struct {
	client *redis.RedisClient
}

func NewRedisAdapter(r *redis.RedisClient) RedisAdapter {
	return &redisWrapper{client: r}
}

func (rw *redisWrapper) SetPopularChat(key string, value string) error {
	// wrap the call and provide default TTL
	return rw.client.SetPopularChat(key, value, 24*time.Hour)
}

func (rw *redisWrapper) GetPopularChat(key string) (string, error) {
	return rw.client.GetPopularChat(key)
}
