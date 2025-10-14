package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Repository interface {
	Push(ctx context.Context, n Notification) error
	List(ctx context.Context, userID string, limit int64) ([]Notification, error)
	MarkRead(ctx context.Context, userID, notifID string) error
}

type redisRepo struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisRepository(rdb *redis.Client) Repository {
	return &redisRepo{rdb: rdb, ttl: 30 * 24 * time.Hour}
}

func key(userID string) string { return fmt.Sprintf("notif:%s", userID) }

func (r *redisRepo) Push(ctx context.Context, n Notification) error {
	b, _ := json.Marshal(n)
	pipe := r.rdb.TxPipeline()
	pipe.LPush(ctx, key(n.UserID), b)
	pipe.Expire(ctx, key(n.UserID), r.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisRepo) List(ctx context.Context, userID string, limit int64) ([]Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	vals, err := r.rdb.LRange(ctx, key(userID), 0, limit-1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Notification, 0, len(vals))
	for _, v := range vals {
		var n Notification
		if json.Unmarshal([]byte(v), &n) == nil {
			out = append(out, n)
		}
	}
	return out, nil
}

func (r *redisRepo) MarkRead(ctx context.Context, userID, notifID string) error {
	items, err := r.List(ctx, userID, 200)
	if err != nil {
		return err
	}
	r.rdb.Del(ctx, key(userID))
	for i := range items {
		if items[i].ID == notifID {
			items[i].Read = true
		}
		_ = r.Push(ctx, items[len(items)-1-i])
	}
	return nil
}
