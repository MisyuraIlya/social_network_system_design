package redisx

import (
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func OpenFromEnv() *redis.Client {
	host := getenv("REDIS_HOST", "redis-feed")
	port := getenv("REDIS_PORT", "6379")
	addr := fmt.Sprintf("%s:%s", host, port)
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return rdb
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
