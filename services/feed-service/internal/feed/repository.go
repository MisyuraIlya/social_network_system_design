package feed

import (
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
)

type Repository interface {
	SaveFeedItem(item FeedItem) error
	GetFeedItemsByUserID(userID string) ([]FeedItem, error)
	SaveFeedItemWithLimit(item FeedItem, limit int) error
}

type redisRepository struct {
	pool *redis.Pool
}

func NewRepository(pool *redis.Pool) Repository {
	return &redisRepository{pool: pool}
}

func (r *redisRepository) SaveFeedItem(item FeedItem) error {
	return r.SaveFeedItemWithLimit(item, 10)
}

func (r *redisRepository) SaveFeedItemWithLimit(item FeedItem, limit int) error {
	conn := r.pool.Get()
	defer conn.Close()

	b, err := json.Marshal(item)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("feed:%s", item.UserID)
	if _, err = conn.Do("LPUSH", key, b); err != nil {
		return err
	}

	_, err = conn.Do("LTRIM", key, 0, limit-1)
	return err
}

func (r *redisRepository) GetFeedItemsByUserID(userID string) ([]FeedItem, error) {
	conn := r.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("feed:%s", userID)
	values, err := redis.Values(conn.Do("LRANGE", key, 0, 9))
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
