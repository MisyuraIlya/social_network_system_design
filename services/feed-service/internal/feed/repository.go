package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyAuthorPostsFmt = "author_posts:%s"
	keyUsersFeedFmt   = "users_feed:%s"
	keyCelebFeedFmt   = "celebrities_feed:%s"

	maxPerAuthor = 500
	maxHomeSize  = 1000
)

type Repository interface {
	HandlePostEvent(ctx context.Context, ev PostEvent) error
	GetAuthorFeed(ctx context.Context, authorID string, limit, offset int) ([]FeedEntry, error)
	StoreHomeFeed(ctx context.Context, userID string, entries []FeedEntry) error
	GetHomeFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error)
}

type repo struct {
	rdb *redis.Client
}

func NewRepository(rdb *redis.Client) Repository { return &repo{rdb: rdb} }

func (r *repo) authorKey(uid string) string   { return fmt.Sprintf(keyAuthorPostsFmt, uid) }
func (r *repo) userFeedKey(uid string) string { return fmt.Sprintf(keyUsersFeedFmt, uid) }

func (r *repo) HandlePostEvent(ctx context.Context, ev PostEvent) error {
	entry := FeedEntry{
		PostID:    ev.ID,
		AuthorID:  ev.UserID,
		MediaURL:  ev.MediaURL,
		Snippet:   ev.Description,
		Tags:      ev.Tags,
		CreatedAt: ev.CreatedAt,
		Score:     float64(ev.CreatedAt.Unix()),
	}
	b, _ := json.Marshal(entry)
	pipe := r.rdb.TxPipeline()
	pipe.LPush(ctx, r.authorKey(ev.UserID), b)
	pipe.LTrim(ctx, r.authorKey(ev.UserID), 0, maxPerAuthor-1)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *repo) GetAuthorFeed(ctx context.Context, authorID string, limit, offset int) ([]FeedEntry, error) {
	raws, err := r.rdb.LRange(ctx, r.authorKey(authorID), int64(offset), int64(offset+limit-1)).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	out := make([]FeedEntry, 0, len(raws))
	for _, s := range raws {
		var e FeedEntry
		if json.Unmarshal([]byte(s), &e) == nil {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *repo) StoreHomeFeed(ctx context.Context, userID string, entries []FeedEntry) error {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Score > entries[j].Score })
	if len(entries) > maxHomeSize {
		entries = entries[:maxHomeSize]
	}
	key := r.userFeedKey(userID)
	pipe := r.rdb.TxPipeline()
	pipe.Del(ctx, key)
	for _, e := range entries {
		b, _ := json.Marshal(e)
		pipe.RPush(ctx, key, b)
	}
	pipe.Expire(ctx, key, 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *repo) GetHomeFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error) {
	key := r.userFeedKey(userID)
	raws, err := r.rdb.LRange(ctx, key, int64(offset), int64(offset+limit-1)).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	out := make([]FeedEntry, 0, len(raws))
	for _, s := range raws {
		var e FeedEntry
		if json.Unmarshal([]byte(s), &e) == nil {
			out = append(out, e)
		}
	}
	return out, nil
}
