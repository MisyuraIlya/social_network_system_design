package redisx

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct{ R *redis.Client }

func NewClientFromEnv() *Client {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "redis-message"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	addr := host + ":" + port
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 0})
	return &Client{R: rdb}
}

// Popular chats via sorted set
const popularKey = "popular_chats"

func (c *Client) IncPopular(ctx context.Context, chatID int64) {
	_ = c.R.ZIncrBy(ctx, popularKey, 1, strconv.FormatInt(chatID, 10)).Err()
	_ = c.R.Expire(ctx, popularKey, 24*time.Hour).Err()
}
func (c *Client) TopPopular(ctx context.Context, n int64) ([]int64, error) {
	items, err := c.R.ZRevRange(ctx, popularKey, 0, n-1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(items))
	for _, s := range items {
		if v, e := strconv.ParseInt(s, 10, 64); e == nil {
			out = append(out, v)
		}
	}
	return out, nil
}
