services/feedback-service/cmd/app/main.go
package main

import (
	"context"
	"feedback-gateway/internal/comment"
	"feedback-gateway/internal/like"
	"feedback-gateway/internal/migrate"
	"feedback-gateway/internal/shared/db"
	"feedback-gateway/internal/shared/httpx"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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

	// Postgres
	store := db.OpenFromEnv()

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: "",
		DB:       0,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}

	// Auto migrate
	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err := migrate.AutoMigrateAll(store); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}

	likeRepo := like.NewRepository(store, rdb)
	likeSvc := like.NewService(likeRepo)

	commentRepo := comment.NewRepository(store, rdb)
	commentSvc := comment.NewService(commentRepo)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	lh := like.NewHandler(likeSvc)
	mux.Handle("GET /posts/{post_id}/likes", httpx.Wrap(lh.GetLikes))

	ch := comment.NewHandler(commentSvc)
	mux.Handle("GET /posts/{post_id}/comments", httpx.Wrap(ch.ListByPost))
	mux.Handle("GET /posts/{post_id}/counts", httpx.Wrap(ch.GetCounts))

	protect := func(pattern string, h http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(h))
	}
	protect("POST /posts/{post_id}/like", httpx.Wrap(lh.Like))
	protect("DELETE /posts/{post_id}/like", httpx.Wrap(lh.Unlike))

	protect("POST /posts/{post_id}/comments", httpx.Wrap(ch.Create))
	protect("DELETE /comments/{comment_id}", httpx.Wrap(ch.DeleteMine))

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
		addr = ":8084"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           otelhttp.NewHandler(mux, "http.server"),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	log.Printf("feedback-service listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}


services/feedback-service/internal/shared/db/db.go
package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct{ DB *gorm.DB }

func OpenFromEnv() *Store {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, pass, name, port,
	)

	var last error
	var g *gorm.DB
	for i := 0; i < 8; i++ {
		g, last = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if last == nil {
			sqlDB, _ := g.DB()
			sqlDB.SetMaxOpenConns(40)
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetConnMaxLifetime(30 * time.Minute)
			return &Store{DB: g}
		}
		time.Sleep(time.Duration(1<<i) * time.Second)
	}
	log.Fatalf("db open failed: %v", last)
	return nil
}


services/feedback-service/internal/shared/httpx/httpx.go
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"feedback-gateway/internal/shared/jwt"
	"net/http"
	"strconv"
	"strings"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) error

func Wrap(fn HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			code := http.StatusBadRequest
			if errors.Is(err, ErrUnauthorized) {
				code = http.StatusUnauthorized
			}
			WriteJSON(w, map[string]any{"error": err.Error()}, code)
		}
	})
}

func Decode[T any](r *http.Request) (T, error) {
	var t T
	err := json.NewDecoder(r.Body).Decode(&t)
	return t, err
}

func WriteJSON(w http.ResponseWriter, v any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

var (
	ctxUserIDKey    = "httpx.user_id"
	ErrUnauthorized = errors.New("unauthorized")
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			WriteJSON(w, map[string]any{"error": "unauthorized", "reason": "missing bearer"}, http.StatusUnauthorized)
			return
		}
		tok := strings.TrimSpace(h[7:])
		uid, err := jwt.Parse(tok)
		if err != nil || uid == "" {
			WriteJSON(w, map[string]any{"error": "unauthorized", "reason": "bad token"}, http.StatusUnauthorized)
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


services/feedback-service/internal/shared/jwt/jwt.go
package jwt

import (
	"errors"
	"os"
	"time"

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
		return "", errors.New("no subject")
	}
	if exp, ok := mc["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return "", errors.New("token expired")
	}
	return uid, nil
}


services/feedback-service/internal/shared/validate/validate.go
package validate

import "github.com/go-playground/validator/v10"

var v = validator.New()

func Struct(s any) error { return v.Struct(s) }


services/feedback-service/internal/migrate/migrate.go
package migrate

import (
	"feedback-gateway/internal/comment"
	"feedback-gateway/internal/like"
	"feedback-gateway/internal/shared/db"
)

func AutoMigrateAll(store *db.Store) error {
	return store.DB.AutoMigrate(
		&like.PostLike{}, &like.PostLikesSum{},
		&comment.PostComment{}, &comment.PostCommentsSum{},
	)
}


services/feedback-service/internal/like/handler.go
package like

import (
	"feedback-gateway/internal/shared/httpx"
	"net/http"
	"strconv"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Like(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	count, err := h.svc.Like(uid, pid)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"post_id": pid, "likes": count, "liked_by_me": true}, http.StatusOK)
	return nil
}

func (h *Handler) Unlike(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	count, err := h.svc.Unlike(uid, pid)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"post_id": pid, "likes": count, "liked_by_me": false}, http.StatusOK)
	return nil
}

func (h *Handler) GetLikes(w http.ResponseWriter, r *http.Request) error {
	uid, _ := httpx.UserFromCtx(r)
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	count, liked, err := h.svc.Get(pid, uid)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"post_id": pid, "likes": count, "liked_by_me": liked}, http.StatusOK)
	return nil
}

services/feedback-service/internal/like/like.go
package like

import "time"

type PostLikesSum struct {
	PostID     uint64 `gorm:"primaryKey" json:"post_id"`
	LikesCount int64  `json:"likes_count"`
	UpdatedAt  time.Time
}

type PostLike struct {
	PostID    uint64 `gorm:"primaryKey;index" json:"post_id"`
	UserID    string `gorm:"primaryKey;size:64;index" json:"user_id"`
	CreatedAt time.Time
}


services/feedback-service/internal/like/repository.go
package like

import (
	"context"
	"feedback-gateway/internal/shared/db"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Like(uid string, postID uint64) (int64, error)
	Unlike(uid string, postID uint64) (int64, error)
	GetCount(postID uint64, forUID string) (int64, bool, error)
}

type repo struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewRepository(s *db.Store, r *redis.Client) Repository {
	return &repo{db: s.DB, rdb: r}
}

func likeKey(postID uint64) string { return fmt.Sprintf("fb:likes:%d", postID) }

func (r *repo) Like(uid string, postID uint64) (int64, error) {
	ctx := context.Background()
	if err := r.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&PostLike{PostID: postID, UserID: uid}).Error; err != nil {
		return 0, err
	}
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "post_id"}},
		DoUpdates: clause.Assignments(map[string]any{"likes_count": gorm.Expr("post_likes_sums.likes_count + EXCLUDED.likes_count")}),
	}).Create(&PostLikesSum{PostID: postID, LikesCount: 1}).Error; err != nil {
		return 0, err
	}
	n, _ := r.rdb.Incr(ctx, likeKey(postID)).Result()
	if n <= 1 {
		var agg PostLikesSum
		if err := r.db.First(&agg, "post_id = ?", postID).Error; err == nil {
			_ = r.rdb.Set(ctx, likeKey(postID), agg.LikesCount, 0).Err()
			n = agg.LikesCount
		}
	}
	return n, nil
}

func (r *repo) Unlike(uid string, postID uint64) (int64, error) {
	ctx := context.Background()
	if err := r.db.Delete(&PostLike{}, "post_id=? AND user_id=?", postID, uid).Error; err != nil {
		return 0, err
	}
	if err := r.db.Exec(
		"UPDATE post_likes_sums SET likes_count = GREATEST(likes_count-1,0) WHERE post_id = ?",
		postID,
	).Error; err != nil {
		return 0, err
	}
	n, _ := r.rdb.Decr(ctx, likeKey(postID)).Result()
	if n < 0 {
		_ = r.rdb.Set(ctx, likeKey(postID), 0, 0).Err()
		n = 0
	}
	return n, nil
}

func (r *repo) GetCount(postID uint64, forUID string) (int64, bool, error) {
	ctx := context.Background()
	val, err := r.rdb.Get(ctx, likeKey(postID)).Int64()
	if err != nil {
		var agg PostLikesSum
		if e := r.db.First(&agg, "post_id = ?", postID).Error; e == nil {
			val = agg.LikesCount
			_ = r.rdb.Set(ctx, likeKey(postID), val, 0).Err()
		} else if e == gorm.ErrRecordNotFound {
			val = 0
		} else {
			return 0, false, e
		}
	}
	var exists int64
	if err := r.db.Model(&PostLike{}).
		Where("post_id = ? AND user_id = ?", postID, forUID).
		Count(&exists).Error; err != nil {
		return 0, false, err
	}
	return val, exists > 0, nil
}


services/feedback-service/internal/like/service.go
package like

type Service interface {
	Like(uid string, postID uint64) (int64, error)
	Unlike(uid string, postID uint64) (int64, error)
	Get(postID uint64, uid string) (int64, bool, error)
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Like(uid string, postID uint64) (int64, error)   { return s.repo.Like(uid, postID) }
func (s *service) Unlike(uid string, postID uint64) (int64, error) { return s.repo.Unlike(uid, postID) }
func (s *service) Get(postID uint64, uid string) (int64, bool, error) {
	return s.repo.GetCount(postID, uid)
}


services/feedback-service/internal/comment/comment.go
package comment

import "time"

type PostCommentsSum struct {
	PostID        uint64 `gorm:"primaryKey" json:"post_id"`
	CommentsCount int64  `json:"comments_count"`
	UpdatedAt     time.Time
}

type PostComment struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	PostID    uint64    `gorm:"index" json:"post_id"`
	UserID    string    `gorm:"size:64;index" json:"user_id"`
	ReplyID   *uint64   `json:"reply_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateReq struct {
	Text    string  `json:"text" validate:"required"`
	ReplyID *uint64 `json:"reply_id"`
}


services/feedback-service/internal/comment/handler.go
package comment

import (
	"feedback-gateway/internal/like"
	"feedback-gateway/internal/shared/httpx"
	"feedback-gateway/internal/shared/validate"
	"net/http"
	"strconv"
)

type Handler struct {
	svc     Service
	likeSvc like.Service
}

func NewHandler(s Service) *Handler                { return &Handler{svc: s} }
func (h *Handler) WithLikeService(ls like.Service) { h.likeSvc = ls }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	in, err := httpx.Decode[CreateReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	c, err := h.svc.Create(uid, pid, in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, c, http.StatusCreated)
	return nil
}

func (h *Handler) DeleteMine(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseUint(r.PathValue("comment_id"), 10, 64)
	if err := h.svc.DeleteMine(uid, cid); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) ListByPost(w http.ResponseWriter, r *http.Request) error {
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.ListByPost(pid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{
		"items": items, "limit": limit, "offset": offset,
	}, http.StatusOK)
	return nil
}

func (h *Handler) GetCounts(w http.ResponseWriter, r *http.Request) error {
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	cCount, err := h.svc.CommentCount(pid)
	if err != nil {
		return err
	}
	var lCount int64
	if h.likeSvc != nil {
		l, _, e := h.likeSvc.Get(pid, "")
		if e == nil {
			lCount = l
		}
	}
	httpx.WriteJSON(w, map[string]any{"post_id": pid, "likes": lCount, "comments": cCount}, http.StatusOK)
	return nil
}


services/feedback-service/internal/comment/repository.go
package comment

import (
	"context"
	"feedback-gateway/internal/shared/db"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(uid string, postID uint64, in CreateReq) (*PostComment, error)
	DeleteMine(uid string, commentID uint64) error
	ListByPost(postID uint64, limit, offset int) ([]PostComment, error)
	Counts(postID uint64) (likes int64, comments int64, err error)
	IncSum(postID uint64, delta int) error
}

type repo struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewRepository(s *db.Store, r *redis.Client) Repository {
	return &repo{db: s.DB, rdb: r}
}

func ckey(postID uint64) string { return fmt.Sprintf("fb:comments:%d", postID) }

func (r *repo) Create(uid string, postID uint64, in CreateReq) (*PostComment, error) {
	pc := &PostComment{PostID: postID, UserID: uid, ReplyID: in.ReplyID, Text: in.Text}
	if err := r.db.Create(pc).Error; err != nil {
		return nil, err
	}
	_ = r.IncSum(postID, +1)
	return pc, nil
}

func (r *repo) DeleteMine(uid string, commentID uint64) error {
	var c PostComment
	if err := r.db.First(&c, "id = ? AND user_id = ?", commentID, uid).Error; err != nil {
		return err
	}
	if err := r.db.Delete(&PostComment{}, "id = ?", commentID).Error; err != nil {
		return err
	}
	return r.IncSum(c.PostID, -1)
}

func (r *repo) IncSum(postID uint64, delta int) error {
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "post_id"}},
		DoUpdates: clause.Assignments(map[string]any{"comments_count": gorm.Expr("post_comments_sums.comments_count + EXCLUDED.comments_count")}),
	}).Create(&PostCommentsSum{PostID: postID, CommentsCount: int64(delta)}).Error; err != nil {
		return err
	}
	ctx := context.Background()
	if delta > 0 {
		_, _ = r.rdb.Incr(ctx, ckey(postID)).Result()
	} else {
		_, _ = r.rdb.Decr(ctx, ckey(postID)).Result()
	}
	return nil
}

func (r *repo) ListByPost(postID uint64, limit, offset int) ([]PostComment, error) {
	var out []PostComment
	err := r.db.Where("post_id = ?", postID).
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

func (r *repo) Counts(postID uint64) (int64, int64, error) {
	var cs PostCommentsSum
	var comments int64
	if err := r.db.First(&cs, "post_id = ?", postID).Error; err == nil {
		comments = cs.CommentsCount
	} else if err == gorm.ErrRecordNotFound {
		comments = 0
	} else {
		return 0, 0, err
	}
	return 0, comments, nil
}

services/feedback-service/internal/comment/service.go
package comment

type Service interface {
	Create(uid string, postID uint64, in CreateReq) (*PostComment, error)
	DeleteMine(uid string, commentID uint64) error
	ListByPost(postID uint64, limit, offset int) ([]PostComment, error)
	CommentCount(postID uint64) (int64, error)
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Create(uid string, postID uint64, in CreateReq) (*PostComment, error) {
	return s.repo.Create(uid, postID, in)
}
func (s *service) DeleteMine(uid string, commentID uint64) error {
	return s.repo.DeleteMine(uid, commentID)
}
func (s *service) ListByPost(postID uint64, limit, offset int) ([]PostComment, error) {
	return s.repo.ListByPost(postID, limit, offset)
}
func (s *service) CommentCount(postID uint64) (int64, error) {
	_, c, err := s.repo.Counts(postID)
	return c, err
}
