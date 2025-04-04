package feed

import (
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
)

// Repository defines how we interact with our data store (Redis).
type Repository interface {
	SaveFeedItem(item FeedItem) error
	GetFeedItemsByUserID(userID string) ([]FeedItem, error)
}

type redisRepository struct {
	pool *redis.Pool
}

func NewRepository(pool *redis.Pool) Repository {
	return &redisRepository{pool: pool}
}

func (r *redisRepository) SaveFeedItem(item FeedItem) error {
	conn := r.pool.Get()
	defer conn.Close()

	b, err := json.Marshal(item)
	if err != nil {
		return err
	}

	// For example: store feed items in a Redis List per user
	key := fmt.Sprintf("feed:%s", item.UserID)
	_, err = conn.Do("LPUSH", key, b)
	return err
}

func (r *redisRepository) GetFeedItemsByUserID(userID string) ([]FeedItem, error) {
	conn := r.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("feed:%s", userID)
	values, err := redis.Values(conn.Do("LRANGE", key, 0, 50)) // get last 50 items
	if err != nil {
		return nil, err
	}

	var feedItems []FeedItem
	for _, v := range values {
		var item FeedItem
		if err := json.Unmarshal(v.([]byte), &item); err == nil {
			feedItems = append(feedItems, item)
		}
	}
	return feedItems, nil
}
