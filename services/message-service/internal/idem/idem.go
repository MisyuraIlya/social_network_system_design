package idem

import (
	"context"
	"time"

	"message-service/internal/redisx"

	"github.com/redis/go-redis/v9"
)

type Store interface {
	PutNX(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

type redisStore struct{ r *redis.Client }

func New(rdb *redisx.Client) Store {
	return &redisStore{r: rdb.R}
}

func (s *redisStore) PutNX(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return s.r.SetNX(ctx, "idem:"+key, "1", ttl).Result()
}
