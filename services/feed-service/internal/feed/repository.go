package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyAuthorPostsFmt = "author_posts:%s"
	keyUsersFeedFmt   = "users_feed:%s"
	keyCelebFeedFmt   = "celebrities_feed:%s"
	keyCelebSet       = "celebrities:set"
	maxPerAuthor      = 500
	maxHomeSize       = 1000
)

type Repository interface {
	HandlePostEvent(ctx context.Context, ev PostEvent) error
	GetAuthorFeed(ctx context.Context, authorID string, limit, offset int) ([]FeedEntry, error)
	StoreHomeFeed(ctx context.Context, userID string, entries []FeedEntry) error
	GetHomeFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error)

	// Celebrities
	AddCelebrity(ctx context.Context, userID string) error
	RemoveCelebrity(ctx context.Context, userID string) error
	IsCelebrity(ctx context.Context, userID string) (bool, error)
	ListCelebrities(ctx context.Context) ([]string, error)
	GetCelebrityFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error)
}

type repo struct {
	rdb *redis.Client
}

func NewRepository(rdb *redis.Client) Repository { return &repo{rdb: rdb} }

func (r *repo) authorKey(uid string) string    { return fmt.Sprintf(keyAuthorPostsFmt, uid) }
func (r *repo) userFeedKey(uid string) string  { return fmt.Sprintf(keyUsersFeedFmt, uid) }
func (r *repo) celebFeedKey(uid string) string { return fmt.Sprintf(keyCelebFeedFmt, uid) }

var (
	weightLikes = getenvFloat("FEED_SCORE_WEIGHT_LIKES", 3600)
	weightViews = getenvFloat("FEED_SCORE_WEIGHT_VIEWS", 60)
)

func getenvFloat(k string, def float64) float64 {
	if s := os.Getenv(k); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return def
}

func computeScore(createdAt time.Time, likes, views int64) float64 {
	base := float64(createdAt.Unix())
	eng := weightLikes*math.Log1p(float64(likes)) + weightViews*math.Log1p(float64(views))
	return base + eng
}

func (r *repo) HandlePostEvent(ctx context.Context, ev PostEvent) error {
	entry := FeedEntry{
		PostID:    ev.ID,
		AuthorID:  ev.UserID,
		MediaURL:  ev.MediaURL,
		Snippet:   ev.Description,
		Tags:      ev.Tags,
		CreatedAt: ev.CreatedAt,
		Score:     computeScore(ev.CreatedAt, ev.Likes, ev.Views),
	}
	b, _ := json.Marshal(entry)

	pipe := r.rdb.TxPipeline()
	pipe.LPush(ctx, r.authorKey(ev.UserID), b)
	pipe.LTrim(ctx, r.authorKey(ev.UserID), 0, maxPerAuthor-1)

	isCeleb, err := r.IsCelebrity(ctx, ev.UserID)
	if err == nil && isCeleb {
		pipe.LPush(ctx, r.celebFeedKey(ev.UserID), b)
		pipe.LTrim(ctx, r.celebFeedKey(ev.UserID), 0, maxPerAuthor-1)
	}
	_, execErr := pipe.Exec(ctx)
	if execErr != nil {
		return execErr
	}
	return nil
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

// ---- Celebrities ----

func (r *repo) AddCelebrity(ctx context.Context, userID string) error {
	return r.rdb.SAdd(ctx, keyCelebSet, userID).Err()
}

func (r *repo) RemoveCelebrity(ctx context.Context, userID string) error {
	return r.rdb.SRem(ctx, keyCelebSet, userID).Err()
}

func (r *repo) IsCelebrity(ctx context.Context, userID string) (bool, error) {
	n, err := r.rdb.SIsMember(ctx, keyCelebSet, userID).Result()
	return n, err
}

func (r *repo) ListCelebrities(ctx context.Context) ([]string, error) {
	return r.rdb.SMembers(ctx, keyCelebSet).Result()
}

func (r *repo) GetCelebrityFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error) {
	raws, err := r.rdb.LRange(ctx, r.celebFeedKey(userID), int64(offset), int64(offset+limit-1)).Result()
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
