# Project code dump

- Generated: 2025-10-16 16:14:10+0300
- Root: `/home/ilya/projects/social_network_system_design/services/post-service`

cmd/app/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"post-service/internal/kafka"
	"post-service/internal/migrate"
	"post-service/internal/post"
	"post-service/internal/shared/db"
	"post-service/internal/shared/httpx"
	"post-service/internal/tag"

	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	store := db.OpenFromEnv()

	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err := migrate.AutoMigrateAll(store); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}

	kWriter, err := kafka.NewWriter(os.Getenv("KAFKA_BOOTSTRAP_SERVERS"), "posts.created")
	if err != nil {
		log.Fatalf("kafka writer: %v", err)
	}
	defer kWriter.Close()

	tagRepo := tag.NewRepository(store)
	tagSvc := tag.NewService(tagRepo)

	postRepo := post.NewRepository(store)
	postSvc := post.NewService(postRepo, tagSvc, kWriter)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	ph := post.NewHandler(postSvc)
	mux.Handle("GET /posts/{post_id}", httpx.Wrap(ph.GetByID))
	mux.Handle("GET /users/{user_id}/posts", httpx.Wrap(ph.ListByUser))

	protect := func(pattern string, h http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(h))
	}

	protect("POST /posts", httpx.Wrap(ph.Create))
	protect("POST /posts/{post_id}/view", httpx.Wrap(ph.AddView))
	protect("POST /posts/upload", httpx.Wrap(ph.UploadAndCreate))

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
		addr = ":8082"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           otelhttp.NewHandler(mux, "http.server"),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	log.Printf("post-service listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

internal/feedback/client.go
package feedback

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const DefaultTimeout = 3 * time.Second

type Client struct {
	base string
	hc   *http.Client
}

func NewClient(base string) *Client {
	if base == "" {
		base = getenv("FEEDBACK_SERVICE_URL", "http://feedback-service:8084")
	}
	return &Client{
		base: base,
		hc:   &http.Client{Timeout: DefaultTimeout},
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func (c *Client) GetCounts(ctx context.Context, postID uint64) (likes int64, comments int64, err error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/posts/%d/counts", c.base, postID), nil)
	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return 0, 0, fmt.Errorf("feedback-service status %d", resp.StatusCode)
	}
	var out struct {
		PostID   uint64 `json:"post_id"`
		Likes    int64  `json:"likes"`
		Comments int64  `json:"comments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, 0, err
	}
	return out.Likes, out.Comments, nil
}

internal/kafka/producer.go
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	kgo "github.com/segmentio/kafka-go"
)

type Writer interface {
	WriteJSON(ctx context.Context, v any) error
	Close() error
}

type writer struct {
	w *kgo.Writer
}

func NewWriter(bootstrapServers, topic string) (Writer, error) {
	addr := "kafka:9092"
	if strings.TrimSpace(bootstrapServers) != "" {
		addr = bootstrapServers
	}
	w := &kgo.Writer{
		Addr:         kgo.TCP(addr),
		Topic:        topic,
		Balancer:     &kgo.LeastBytes{},
		RequiredAcks: kgo.RequireOne,
		Async:        false,
		BatchTimeout: 50 * time.Millisecond,
	}
	return &writer{w: w}, nil
}

func (wr *writer) WriteJSON(ctx context.Context, v any) error {
	b, err := jsonMarshal(v)
	if err != nil {
		return err
	}
	msg := kgo.Message{Value: b}
	return wr.w.WriteMessages(ctx, msg)
}

func (wr *writer) Close() error { return wr.w.Close() }

func jsonMarshal(v any) ([]byte, error) {
	switch t := v.(type) {
	case []byte:
		return t, nil
	default:
		return jsonMarshalStd(v)
	}
}

func jsonMarshalStd(v any) ([]byte, error) {
	type json = struct{}
	_ = json{}
	return jsonMarshalImpl(v)
}

func jsonMarshalImpl(v any) ([]byte, error) { return json.Marshal(v) }

var _ = fmt.Sprintf

internal/migrate/migrate.go
package migrate

import (
	"post-service/internal/post"
	"post-service/internal/shared/db"
	"post-service/internal/tag"
)

func AutoMigrateAll(store *db.Store) error {
	return store.Base.AutoMigrate(
		&post.Post{},
		&post.PostTag{},
		&tag.Tag{},
	)
}

internal/post/handler.go
package post

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"post-service/internal/feedback"
	"post-service/internal/shared/httpx"
	"post-service/internal/shared/validate"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[CreateReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	p, err := h.svc.Create(uid, in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, p, http.StatusCreated)
	return nil
}

func (h *Handler) UploadAndCreate(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	if err := r.ParseMultipartForm(20 << 20); err != nil { // 20MB
		return err
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer file.Close()

	description := strings.TrimSpace(r.FormValue("description"))
	tags := strings.Split(strings.TrimSpace(r.FormValue("tags")), ",")
	if len(tags) == 1 && tags[0] == "" {
		tags = nil
	}

	p, err := h.svc.UploadAndCreate(uid, hdr.Filename, file, description, tags)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, p, http.StatusCreated)
	return nil
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	p, err := h.svc.GetByID(id)
	if err != nil {
		return err
	}

	// Enrich counts from feedback-service
	ctx, cancel := context.WithTimeout(r.Context(), feedback.DefaultTimeout)
	defer cancel()
	fb := feedback.NewClient("")
	likes, comments, _ := fb.GetCounts(ctx, p.ID)

	out := map[string]any{
		"id":          p.ID,
		"user_id":     p.UserID,
		"description": p.Description,
		"media":       p.MediaURL,
		"views":       p.Views,
		"likes":       likes,
		"comments":    comments,
		"created_at":  p.CreatedAt,
		"updated_at":  p.UpdatedAt,
	}
	httpx.WriteJSON(w, out, http.StatusOK)
	return nil
}

func (h *Handler) ListByUser(w http.ResponseWriter, r *http.Request) error {
	uid := r.PathValue("user_id")
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.ListByUser(uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) AddView(w http.ResponseWriter, r *http.Request) error {
	id, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	if err := h.svc.AddView(id); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

internal/post/post.go
package post

import "time"

type Post struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"index;size:64" json:"user_id"`
	Description string    `json:"description"`
	MediaURL    string    `gorm:"size:512" json:"media"`
	Likes       uint64    `json:"-"` // hidden; feedback-service is the source of truth
	Views       uint64    `json:"views"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PostTag struct {
	PostID uint64 `gorm:"primaryKey"`
	TagID  uint64 `gorm:"primaryKey"`
}

type CreateReq struct {
	Description string   `json:"description" validate:"required"`
	MediaURL    string   `json:"media_url"`
	Tags        []string `json:"tags"`
}

type LikeReq struct {
}

internal/post/repository.go
package post

import (
	"errors"

	"post-service/internal/shared/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(p *Post) (*Post, error)
	GetByID(id uint64) (*Post, error)
	ListByUser(userID string, limit, offset int) ([]Post, error)
	AttachTags(postID uint64, tagIDs []uint64) error
	IncView(postID uint64) error
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(p *Post) (*Post, error) {
	if err := r.store.Base.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (r *repo) GetByID(id uint64) (*Post, error) {
	var p Post
	if err := r.store.Base.First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repo) ListByUser(userID string, limit, offset int) ([]Post, error) {
	var out []Post
	err := r.store.Base.
		Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

func (r *repo) AttachTags(postID uint64, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	items := make([]PostTag, 0, len(tagIDs))
	for _, id := range tagIDs {
		items = append(items, PostTag{PostID: postID, TagID: id})
	}
	return r.store.Base.Clauses(clause.OnConflict{DoNothing: true}).Create(&items).Error
}

func (r *repo) IncView(postID uint64) error {
	res := r.store.Base.Model(&Post{}).Where("id = ?", postID).UpdateColumn("views", gorm.Expr("views + 1"))
	return res.Error
}

var _ = errors.New

internal/post/service.go
package post

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"post-service/internal/kafka"
	"post-service/internal/shared/validate"
	"post-service/internal/tag"
)

type Service interface {
	Create(uid string, in CreateReq) (*Post, error)
	GetByID(id uint64) (*Post, error)
	ListByUser(userID string, limit, offset int) ([]Post, error)
	AddView(postID uint64) error
	UploadAndCreate(uid string, filename string, file io.Reader, description string, tags []string) (*Post, error)
}

type service struct {
	repo  Repository
	tags  tag.Service
	kafka kafka.Writer
}

func NewService(r Repository, t tag.Service, kw kafka.Writer) Service {
	return &service{repo: r, tags: t, kafka: kw}
}

func (s *service) Create(uid string, in CreateReq) (*Post, error) {
	if err := validate.Struct(in); err != nil {
		return nil, err
	}
	p := &Post{
		UserID: uid, Description: in.Description, MediaURL: in.MediaURL,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	out, err := s.repo.Create(p)
	if err != nil {
		return nil, err
	}
	tgs, err := s.tags.Ensure(in.Tags)
	if err != nil {
		return nil, err
	}
	if len(tgs) > 0 {
		ids := make([]uint64, 0, len(tgs))
		for _, t := range tgs {
			ids = append(ids, t.ID)
		}
		if err := s.repo.AttachTags(out.ID, ids); err != nil {
			return nil, err
		}
	}
	_ = s.kafka.WriteJSON(context.Background(), map[string]any{
		"id":          out.ID,
		"user_id":     out.UserID,
		"description": out.Description,
		"media_url":   out.MediaURL,
		"tags":        in.Tags,
		"created_at":  out.CreatedAt,
	})
	return out, nil
}

func (s *service) GetByID(id uint64) (*Post, error) { return s.repo.GetByID(id) }

func (s *service) ListByUser(userID string, limit, offset int) ([]Post, error) {
	return s.repo.ListByUser(userID, limit, offset)
}

func (s *service) AddView(postID uint64) error { return s.repo.IncView(postID) }

func (s *service) UploadAndCreate(uid, filename string, file io.Reader, description string, tags []string) (*Post, error) {
	mediaURL, err := uploadToMediaService(filename, file)
	if err != nil {
		return nil, err
	}
	return s.Create(uid, CreateReq{Description: description, MediaURL: mediaURL, Tags: tags})
}

func uploadToMediaService(filename string, r io.Reader) (string, error) {
	base := os.Getenv("MEDIA_SERVICE_URL")
	if base == "" {
		base = "http://media-service:8088"
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", filename)
	if _, err := io.Copy(fw, r); err != nil {
		return "", err
	}
	_ = w.Close()

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/media/upload", base), &body)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media-service: %s", string(b))
	}
	type out struct {
		URL string `json:"url"`
	}
	var o out
	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return "", err
	}
	return o.URL, nil
}

internal/shared/db/simple.go
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct{ Base *gorm.DB }

func OpenFromEnv() *Store {
	host := getenv("DB_HOST", "post-db")
	user := getenv("DB_USER", "post")
	pass := getenv("DB_PASSWORD", "postpass")
	name := getenv("DB_NAME", "post_db")
	port := getenv("DB_PORT", "5432")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, name,
	)

	var base *gorm.DB
	var err error
	sleep := time.Second
	for i := 0; i < 8; i++ {
		base, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			sqlDB, _ := base.DB()
			if pingWithTimeout(sqlDB, 2*time.Second) == nil {
				break
			}
		}
		time.Sleep(sleep)
		if sleep < 8*time.Second {
			sleep *= 2
		}
	}
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	sqlDB, _ := base.DB()
	sqlDB.SetMaxOpenConns(40)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return &Store{Base: base}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func pingWithTimeout(sqlDB *sql.DB, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- sqlDB.Ping() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("db ping timeout after %s", timeout)
	}
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

	"post-service/internal/shared/jwt"
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
		uid, _, err := jwt.Parse(tok)
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

internal/shared/jwt/jwt.go
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

func Make(userID string) (string, error) {
	claims := jw.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	return jw.NewWithClaims(jw.SigningMethodHS256, claims).SignedString(secret())
}

func Parse(tok string) (string, int, error) {
	t, err := jw.Parse(tok, func(t *jw.Token) (any, error) { return secret(), nil })
	if err != nil || !t.Valid {
		return "", 0, errors.New("invalid token")
	}
	mc, ok := t.Claims.(jw.MapClaims)
	if !ok {
		return "", 0, errors.New("bad claims")
	}
	uid, _ := mc["sub"].(string)
	return uid, 0, nil
}

internal/shared/validate/validate.go
package validate

import "github.com/go-playground/validator/v10"

var v = validator.New()

func Struct(s any) error { return v.Struct(s) }

internal/tag/repository.go
package tag

import (
	"post-service/internal/shared/db"

	"gorm.io/gorm"
)

type Repository interface {
	FirstOrCreateByName(name string) (*Tag, error)
	FindByNames(names []string) ([]Tag, error)
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) FirstOrCreateByName(name string) (*Tag, error) {
	t := &Tag{Name: name}
	if err := r.store.Base.FirstOrCreate(t, "name = ?", name).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func (r *repo) FindByNames(names []string) ([]Tag, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var out []Tag
	err := r.store.Base.Where("name IN ?", names).Find(&out).Error
	return out, err
}

var _ = gorm.ErrRecordNotFound

internal/tag/service.go
package tag

type Service interface {
	Ensure(names []string) ([]Tag, error)
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Ensure(names []string) ([]Tag, error) {
	out := make([]Tag, 0, len(names))
	seen := map[string]struct{}{}
	for _, n := range names {
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		t, err := s.repo.FirstOrCreateByName(n)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, nil
}

internal/tag/tag.go
package tag

import "time"

type Tag struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:120" json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

