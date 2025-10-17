# Project code dump

- Generated: 2025-10-17 11:38:01+0300
- Root: `/home/spetsar/projects/social_network_system_design/services/feed-service`

cmd/app/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"feed-service/internal/feed"
	"feed-service/internal/kafka"
	"feed-service/internal/ratelimit"
	"feed-service/internal/shared/httpx"
	"feed-service/internal/shared/redisx"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func atoiDef(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func initOTEL(ctx context.Context) func(context.Context) error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4318"
	}
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	if err != nil {
		log.Fatalf("otel exporter: %v", err)
	}
	res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(os.Getenv("OTEL_SERVICE_NAME")),
		attribute.String("deployment.environment", os.Getenv("ENV")),
	))
	ratio := 1.0
	if s := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); s != "" {
		if f, e := strconv.ParseFloat(s, 64); e == nil && f >= 0 && f <= 1 {
			ratio = f
		}
	}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(ratio))),
		trace.WithBatcher(exp),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	return tp.Shutdown
}

func main() {
	ctx := context.Background()
	shutdown := initOTEL(ctx)
	defer func() {
		c, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = shutdown(c)
	}()

	// Redis
	rdb := redisx.OpenFromEnv()
	defer func(rdb *redis.Client) { _ = rdb.Close() }(rdb)

	// Rate limiter (Redis-backed)
	limiter := ratelimit.New(rdb)
	rebuildLimit := func(next http.Handler) http.Handler {
		return limiter.LimitHTTP(1, 60*time.Second, func(r *http.Request) (string, error) {
			return httpx.UserFromCtx(r)
		}, next)
	}

	// Repo & Service
	repo := feed.NewRepository(rdb)
	svc := feed.NewService(
		repo,
		feed.WithUserServiceBase(os.Getenv("USER_SERVICE_URL")),
		feed.WithPostServiceBase(os.Getenv("POST_SERVICE_URL")), // optional enrichment endpoint
		feed.WithDefaultFeedLimit(atoiDef(os.Getenv("FEED_DEFAULT_LIMIT"), 100)),
	)

	// Kafka consumer
	bootstrap := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if bootstrap == "" {
		bootstrap = "kafka:9092"
	}
	groupID := os.Getenv("KAFKA_GROUP_ID")
	if groupID == "" {
		groupID = "feed-service"
	}
	topic := os.Getenv("POSTS_TOPIC")
	if topic == "" {
		topic = "posts.created"
	}
	go func() {
		if err := kafka.StartConsumer(ctx, bootstrap, topic, groupID, repo.HandlePostEvent); err != nil {
			log.Printf("kafka consumer stopped: %v", err)
		}
	}()

	// HTTP
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	h := feed.NewHandler(svc)

	// Public:
	mux.Handle("GET /users/{user_id}/feed", httpx.Wrap(h.GetAuthorFeed))
	mux.Handle("GET /celebrities/{user_id}/feed", httpx.Wrap(h.GetCelebrityFeed))
	mux.Handle("GET /celebrities", httpx.Wrap(h.ListCelebrities))

	// Protected:
	protect := func(pattern string, handler http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(handler))
	}
	protect("GET /feed", httpx.Wrap(h.GetHomeFeed))
	protect("POST /feed/rebuild", rebuildLimit(httpx.Wrap(h.RebuildHomeFeed)))

	protect("POST /celebrities/{user_id}", httpx.Wrap(h.PromoteCelebrity))
	protect("DELETE /celebrities/{user_id}", httpx.Wrap(h.DemoteCelebrity))

	protect("GET /whoami", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, err := httpx.UserFromCtx(r)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusUnauthorized)
			return
		}
		httpx.WriteJSON(w, map[string]any{"user_id": uid}, http.StatusOK)
	}))

	addr := os.Getenv("APP_PORT")
	if addr == "" {
		addr = ":8083"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           otelhttp.NewHandler(mux, "http.server"),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	log.Printf("feed-service listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

internal/feed/handler.go
package feed

import (
	"net/http"
	"os"

	"feed-service/internal/shared/httpx"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

// Public: feed by author
func (h *Handler) GetAuthorFeed(w http.ResponseWriter, r *http.Request) error {
	uid := r.PathValue("user_id")
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.GetAuthorFeed(r.Context(), uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

// Protected: home feed of the current user
func (h *Handler) GetHomeFeed(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.GetHomeFeed(r.Context(), uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) RebuildHomeFeed(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	bearer := httpx.BearerToken(r)
	limit := httpx.QueryInt(r, "limit", 100)
	if err := h.svc.RebuildHomeFeed(r.Context(), uid, bearer, limit); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

// ---------- Celebrities ----------

// Public: feed by celebrity user_id
func (h *Handler) GetCelebrityFeed(w http.ResponseWriter, r *http.Request) error {
	uid := r.PathValue("user_id")
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.GetCelebrityFeed(r.Context(), uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

// Public: list celebrity IDs (could be cached by clients)
func (h *Handler) ListCelebrities(w http.ResponseWriter, r *http.Request) error {
	ids, err := h.svc.ListCelebrities(r.Context())
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": ids}, http.StatusOK)
	return nil
}

// Protected: promote a user to celebrity set (admin only via X-Admin-Token)
func (h *Handler) PromoteCelebrity(w http.ResponseWriter, r *http.Request) error {
	if os.Getenv("ADMIN_TOKEN") == "" || r.Header.Get("X-Admin-Token") != os.Getenv("ADMIN_TOKEN") {
		return httpx.ErrUnauthorized
	}
	uid := r.PathValue("user_id")
	if uid == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.PromoteCelebrity(r.Context(), uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

// Protected: demote a user from celebrity set (admin only via X-Admin-Token)
func (h *Handler) DemoteCelebrity(w http.ResponseWriter, r *http.Request) error {
	if os.Getenv("ADMIN_TOKEN") == "" || r.Header.Get("X-Admin-Token") != os.Getenv("ADMIN_TOKEN") {
		return httpx.ErrUnauthorized
	}
	uid := r.PathValue("user_id")
	if uid == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.DemoteCelebrity(r.Context(), uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

internal/feed/repository.go
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

internal/feed/service.go
package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Service interface {
	GetAuthorFeed(ctx context.Context, authorID string, limit, offset int) ([]FeedEntry, error)
	GetHomeFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error)
	RebuildHomeFeed(ctx context.Context, userID, bearer string, limit int) error

	// Celebrities
	GetCelebrityFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error)
	PromoteCelebrity(ctx context.Context, userID string) error
	DemoteCelebrity(ctx context.Context, userID string) error
	ListCelebrities(ctx context.Context) ([]string, error)
}

type service struct {
	repo             Repository
	userSvcBase      string
	postSvcBase      string
	defaultFeedLimit int
	httpClient       *http.Client
}

type Option func(*service)

func WithUserServiceBase(base string) Option {
	return func(s *service) { s.userSvcBase = base }
}
func WithDefaultFeedLimit(n int) Option {
	return func(s *service) { s.defaultFeedLimit = n }
}

func WithPostServiceBase(base string) Option {
	return func(s *service) { s.postSvcBase = base }
}

func NewService(r Repository, opts ...Option) Service {
	s := &service{
		repo:             r,
		userSvcBase:      envOr("USER_SERVICE_URL", "http://user-service:8081"),
		defaultFeedLimit: 100,
		httpClient:       &http.Client{Timeout: 5 * time.Second},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func (s *service) GetAuthorFeed(ctx context.Context, authorID string, limit, offset int) ([]FeedEntry, error) {
	return s.repo.GetAuthorFeed(ctx, authorID, limit, offset)
}

func (s *service) GetHomeFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error) {
	return s.repo.GetHomeFeed(ctx, userID, limit, offset)
}

func (s *service) RebuildHomeFeed(ctx context.Context, userID, bearer string, limit int) error {
	if limit <= 0 {
		limit = s.defaultFeedLimit
	}

	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/relationships?type=1&limit=%d&offset=%d", s.userSvcBase, 5000, 0), nil)
	req.Header.Set("Authorization", "Bearer "+bearer)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("user-service call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("user-service: bad status %d", resp.StatusCode)
	}
	var rel struct {
		Items []string `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return err
	}

	all := make([]FeedEntry, 0, limit*2)
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	perAuthor := limit / 5
	if perAuthor < 10 {
		perAuthor = 10
	}
	if perAuthor > 100 {
		perAuthor = 100
	}
	for _, authorID := range rel.Items {
		ents, e := s.repo.GetAuthorFeed(ctx2, authorID, perAuthor, 0)
		if e == nil && len(ents) > 0 {
			all = append(all, ents...)
		}
		// If Redis feed for this author is too small, backfill via post-service
		if len(ents) < perAuthor {
			more, e2 := s.fetchAuthorRecentPosts(ctx2, authorID, perAuthor-len(ents), bearer)
			if e2 == nil && len(more) > 0 {
				all = append(all, more...)
			}
		}
	}

	sort.Slice(all, func(i, j int) bool { return all[i].Score > all[j].Score })
	if len(all) > limit {
		all = all[:limit]
	}
	return s.repo.StoreHomeFeed(ctx, userID, all)
}

func (s *service) fetchAuthorRecentPosts(ctx context.Context, authorID string, limit int, bearer string) ([]FeedEntry, error) {
	if s.postSvcBase == "" {
		return nil, nil
	}
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/users/%s/posts?limit=%d&offset=0", s.postSvcBase, authorID, limit), nil)

	// If the post-service requires auth for user routes, pass through the caller's bearer
	if strings.TrimSpace(bearer) != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post-service status %d", resp.StatusCode)
	}

	var pl postListResp
	if err := json.NewDecoder(resp.Body).Decode(&pl); err != nil {
		return nil, err
	}

	out := make([]FeedEntry, 0, len(pl.Items))
	for _, p := range pl.Items {
		out = append(out, FeedEntry{
			PostID:    p.ID,
			AuthorID:  p.UserID,
			MediaURL:  p.Media,
			Snippet:   p.Description,
			CreatedAt: p.CreatedAt,
			Score:     float64(p.CreatedAt.Unix()),
		})
	}
	return out, nil
}

// ---- Celebrities ----

func (s *service) GetCelebrityFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error) {
	return s.repo.GetCelebrityFeed(ctx, userID, limit, offset)
}

func (s *service) PromoteCelebrity(ctx context.Context, userID string) error {
	return s.repo.AddCelebrity(ctx, userID)
}

func (s *service) DemoteCelebrity(ctx context.Context, userID string) error {
	return s.repo.RemoveCelebrity(ctx, userID)
}

func (s *service) ListCelebrities(ctx context.Context) ([]string, error) {
	return s.repo.ListCelebrities(ctx)
}

internal/feed/types.go
package feed

import "time"

type PostEvent struct {
	ID          int64     `json:"id"`
	UserID      string    `json:"user_id"`
	Description string    `json:"description"`
	MediaURL    string    `json:"media_url"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	Likes       int64     `json:"likes,omitempty"`
	Views       int64     `json:"views,omitempty"`
}

type postListResp struct {
	Items []struct {
		ID          int64     `json:"id"`
		UserID      string    `json:"user_id"`
		Description string    `json:"description"`
		Media       string    `json:"media"`
		CreatedAt   time.Time `json:"created_at"`
	} `json:"items"`
}

type FeedEntry struct {
	PostID    int64     `json:"post_id"`
	AuthorID  string    `json:"author_id"`
	MediaURL  string    `json:"media_url,omitempty"`
	Snippet   string    `json:"snippet,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Score     float64   `json:"score"`
}

internal/kafka/consumer.go
package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"feed-service/internal/feed"

	kf "github.com/segmentio/kafka-go"
)

type PostHandler func(ctx context.Context, ev feed.PostEvent) error

func StartConsumer(ctx context.Context, bootstrap, topic, groupID string, handle PostHandler) error {
	r := kf.NewReader(kf.ReaderConfig{
		Brokers:  strings.Split(bootstrap, ","),
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
		MaxWait:  2 * time.Second,
	})
	defer r.Close()

	log.Printf("kafka consumer started group=%s topic=%s", groupID, topic)

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			return err
		}
		var ev feed.PostEvent
		if err := json.Unmarshal(m.Value, &ev); err != nil {
			log.Printf("kafka: bad payload: %v", err)
			continue
		}
		if err := handle(ctx, ev); err != nil {
			log.Printf("handle post event: %v", err)
		}
	}
}

internal/ratelimit/ratelimit.go
package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"feed-service/internal/shared/httpx"

	"github.com/redis/go-redis/v9"
)

type Limiter struct{ R *redis.Client }

func New(r *redis.Client) *Limiter { return &Limiter{R: r} }

func (l *Limiter) AllowSliding(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, error) {
	k := "rl:" + key
	pipe := l.R.TxPipeline()
	incr := pipe.Incr(ctx, k)
	pipe.Expire(ctx, k, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}
	n := incr.Val()
	return n <= limit, n, nil
}

func (l *Limiter) LimitHTTP(limit int64, window time.Duration, keyFn func(*http.Request) (string, error), next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, err := keyFn(r)
		if err != nil || key == "" {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrUnauthorized, "missing_user")
			return
		}
		ok, n, e := l.AllowSliding(r.Context(), key, limit, window)
		if e != nil {
			httpx.WriteError(w, http.StatusTooManyRequests, fmt.Errorf("rate limiter error"), "rate_limiter_error")
			return
		}
		if !ok {
			httpx.WriteError(w, http.StatusTooManyRequests,
				fmt.Errorf("rate limit exceeded (count=%d, limit=%d)", n, limit),
				"rate_limited")
			return
		}
		next.ServeHTTP(w, r)
	})
}

internal/shared/httpx/httpx.go
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"feed-service/internal/shared/jwt"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) error

type APIError struct {
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
	Status int    `json:"status"`
}

var (
	ctxUserIDKey    = "httpx.user_id"
	ErrUnauthorized = errors.New("unauthorized")
)

func WriteJSON(w http.ResponseWriter, v any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, err error, reason string) {
	if err == nil {
		err = errors.New(http.StatusText(status))
	}
	WriteJSON(w, APIError{Error: err.Error(), Reason: reason, Status: status}, status)
}

func Wrap(fn HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			code := http.StatusBadRequest
			if errors.Is(err, ErrUnauthorized) {
				code = http.StatusUnauthorized
			}
			WriteError(w, code, err, "")
		}
	})
}

func Decode[T any](r *http.Request) (T, error) {
	var t T
	err := json.NewDecoder(r.Body).Decode(&t)
	return t, err
}

func WriteBadRequest(w http.ResponseWriter, err error, reason string) {
	WriteError(w, http.StatusBadRequest, err, reason)
}

func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := BearerToken(r)
		if tok == "" {
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "missing_bearer")
			return
		}
		uid, err := jwt.Parse(tok)
		if err != nil || uid == "" {
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid_token")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserIDKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromCtx(r *http.Request) (string, error) {
	uid, _ := r.Context().Value(ctxUserIDKey).(string)
	if uid == "" {
		return "", ErrUnauthorized
	}
	return uid, nil
}

func QueryInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

internal/shared/jwt/jwt.go
package jwt

import (
	"errors"
	"os"

	jw "github.com/golang-jwt/jwt/v5"
)

func secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("replace-this-with-a-strong-secret")
}

func Parse(tok string) (string, error) {
	t, err := jw.Parse(tok, func(t *jw.Token) (any, error) { return secret(), nil })
	if err != nil || !t.Valid {
		return "", errors.New("invalid token")
	}
	mc, ok := t.Claims.(jw.MapClaims)
	if !ok {
		return "", errors.New("bad claims")
	}
	uid, _ := mc["sub"].(string)
	if uid == "" {
		return "", errors.New("missing sub")
	}
	return uid, nil
}

internal/shared/redisx/redisx.go
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

