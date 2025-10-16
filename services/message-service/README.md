services/message-service/cmd/app/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"message-service/internal/chat"
	"message-service/internal/kafka"
	"message-service/internal/media"
	"message-service/internal/message"
	"message-service/internal/migrate"
	"message-service/internal/redisx"
	"message-service/internal/shared/db"
	"message-service/internal/shared/httpx"

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

	// Postgres
	store := db.OpenFromEnv()

	// Redis
	rds := redisx.NewClientFromEnv()

	// Kafka producer
	kWriter, err := kafka.NewWriter(os.Getenv("KAFKA_BOOTSTRAP_SERVERS"), "messages.created")
	if err != nil {
		log.Fatalf("kafka writer: %v", err)
	}
	defer kWriter.Close()

	// Media client
	mediaCli := media.New(os.Getenv("MEDIA_SERVICE_URL"))

	// Auto-migrate schema
	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err := migrate.AutoMigrateAll(store); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}

	// Wire repos & services
	chatRepo := chat.NewRepository(store)
	chatSvc := chat.NewService(chatRepo, rds)

	msgRepo := message.NewRepository(store)
	msgSvc := message.NewService(msgRepo, chatSvc, rds, kWriter, mediaCli)

	// HTTP
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Public (lookups)
	ch := chat.NewHandler(chatSvc)
	mh := message.NewHandler(msgSvc)

	mux.Handle("GET /chats/{chat_id}", httpx.Wrap(ch.GetByID))

	// Protected
	protect := func(pattern string, h http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(h))
	}

	// Chats
	protect("POST /chats", httpx.Wrap(ch.Create))                          // create chat (1:1 or group)
	protect("GET /chats", httpx.Wrap(ch.ListMine))                         // list my chats
	protect("POST /chats/{chat_id}/join", httpx.Wrap(ch.Join))             // join (add self)
	protect("POST /chats/{chat_id}/add/{user_id}", httpx.Wrap(ch.AddUser)) // add someone
	protect("POST /chats/{chat_id}/leave", httpx.Wrap(ch.Leave))           // leave chat
	mux.Handle("GET /chats/popular", httpx.Wrap(ch.Popular))               // read-only, from Redis

	// Messages
	protect("GET /chats/{chat_id}/messages", httpx.Wrap(mh.ListByChat)) // history
	protect("POST /messages", httpx.Wrap(mh.Send))                      // text-only
	protect("POST /messages/upload", httpx.Wrap(mh.UploadAndSend))      // upload media + send
	protect("POST /messages/{message_id}/seen", httpx.Wrap(mh.MarkSeen))

	// Health/info
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
		addr = ":8085"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           otelhttp.NewHandler(mux, "http.server"),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	log.Printf("message-service listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

services/message-service/internal/shared/db/single.go
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
	host := def(os.Getenv("DB_HOST"), "message-db")
	user := def(os.Getenv("DB_USER"), "notify")
	pass := def(os.Getenv("DB_PASSWORD"), "notifypass")
	name := def(os.Getenv("DB_NAME"), "message_db")
	port := def(os.Getenv("DB_PORT"), "5432")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, name)

	base, err := openWithRetry(dsn, 8, 2*time.Second)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	sqlDB, _ := base.DB()
	sqlDB.SetMaxOpenConns(40)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return &Store{Base: base}
}

func def(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func openWithRetry(dsn string, attempts int, sleep time.Duration) (*gorm.DB, error) {
	var last error
	for i := 1; i <= attempts; i++ {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			if s, e := db.DB(); e == nil && s != nil {
				if perr := pingWithTimeout(s, 2*time.Second); perr == nil {
					return db, nil
				} else {
					last = perr
				}
			} else {
				last = e
			}
		} else {
			last = err
		}
		time.Sleep(sleep)
		if sleep < 8*time.Second {
			sleep *= 2
		}
	}
	return nil, last
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


services/message-service/internal/shared/httpx/httpx.go
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"message-service/internal/shared/jwt"
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


services/message-service/internal/shared/jwt/jwt.go
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

// Parse returns userID from the token (we only need "sub")
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


services/message-service/internal/shared/validate/validate.go
package validate

import "github.com/go-playground/validator/v10"

var v = validator.New()

func Struct(s any) error { return v.Struct(s) }


services/message-service/internal/redisx/cache.go
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

services/message-service/internal/migrate/migrate.go
package migrate

import (
	"message-service/internal/chat"
	"message-service/internal/message"
	"message-service/internal/shared/db"
)

func AutoMigrateAll(store *db.Store) error {
	return store.Base.AutoMigrate(
		&chat.Chat{}, &chat.ChatUser{},
		&message.Message{},
	)
}


services/message-service/internal/message/handler.go
package message

import (
	"io"
	"net/http"
	"strconv"

	"message-service/internal/shared/httpx"
	"message-service/internal/shared/validate"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[SendReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	m, err := h.svc.Send(r.Context(), uid, in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, m, http.StatusCreated)
	return nil
}

func (h *Handler) UploadAndSend(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return err
	}
	f, fh, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer f.Close()
	data, _ := io.ReadAll(f)
	chatID, _ := strconv.ParseInt(r.FormValue("chat_id"), 10, 64)
	text := r.FormValue("text")
	m, err := h.svc.SendWithUpload(r.Context(), uid, chatID, fh.Filename, data, text)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, m, http.StatusCreated)
	return nil
}

func (h *Handler) ListByChat(w http.ResponseWriter, r *http.Request) error {
	_, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	limit := qint(r, "limit", 50)
	offset := qint(r, "offset", 0)
	items, err := h.svc.ListByChat(cid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) MarkSeen(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	mid, _ := strconv.ParseInt(r.PathValue("message_id"), 10, 64)
	if err := h.svc.MarkSeen(mid, uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func qint(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		return def
	}
	return n
}


services/message-service/internal/message/message.go
package message

import "time"

type Message struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	UserID        string    `gorm:"size:64" json:"user_id"`
	ChatID        int64     `gorm:"index" json:"chat_id"`
	Text          string    `json:"text"`
	MediaURL      string    `gorm:"size:512" json:"media_url"`
	IsSeen        bool      `json:"is_seen"`
	SendTime      time.Time `json:"send_time"`
	DeliveredTime time.Time `json:"delivered_time"`
}

type SendReq struct {
	ChatID   int64  `json:"chat_id" validate:"required"`
	Text     string `json:"text"`
	MediaURL string `json:"media_url"`
}

services/message-service/internal/message/repository.go
package message

import (
	"message-service/internal/shared/db"
)

type Repository interface {
	Create(m *Message) (*Message, error)
	MarkSeen(messageID int64, userID string) error
	ListByChat(chatID int64, limit, offset int) ([]Message, error)
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(m *Message) (*Message, error) {
	if err := r.store.Base.Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (r *repo) MarkSeen(messageID int64, userID string) error {
	return r.store.Base.Model(&Message{}).Where("id=? AND user_id=?", messageID, userID).
		Update("is_seen", true).Error
}

func (r *repo) ListByChat(chatID int64, limit, offset int) ([]Message, error) {
	var out []Message
	err := r.store.Base.
		Where("chat_id = ?", chatID).
		Order("id DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

services/message-service/internal/message/service.go
package message

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"message-service/internal/chat"
	"message-service/internal/kafka"
	"message-service/internal/media"
	"message-service/internal/redisx"
)

type Service interface {
	Send(ctx context.Context, userID string, in SendReq) (*Message, error)
	SendWithUpload(ctx context.Context, userID string, chatID int64, fileName string, fileData []byte, text string) (*Message, error)
	MarkSeen(messageID int64, userID string) error
	ListByChat(chatID int64, limit, offset int) ([]Message, error)
}

type service struct {
	repo  Repository
	chats chat.Service
	rds   *redisx.Client
	kafka *kafka.Writer
	media *media.Client
}

func NewService(r Repository, cs chat.Service, rds *redisx.Client, kw *kafka.Writer, mc *media.Client) Service {
	return &service{repo: r, chats: cs, rds: rds, kafka: kw, media: mc}
}

func (s *service) Send(ctx context.Context, userID string, in SendReq) (*Message, error) {
	// ensure chat exists
	if _, err := s.chats.GetByID(in.ChatID); err != nil {
		return nil, err
	}
	m := &Message{
		UserID: userID, ChatID: in.ChatID,
		Text: in.Text, MediaURL: in.MediaURL,
		SendTime: time.Now(),
	}
	res, err := s.repo.Create(m)
	if err != nil {
		return nil, err
	}

	// side effects: popularity + kafka event
	s.chats.IncPopular(ctx, in.ChatID)
	_ = s.emit(res)

	return res, nil
}

func (s *service) SendWithUpload(ctx context.Context, userID string, chatID int64, fileName string, data []byte, text string) (*Message, error) {
	url, err := s.media.Upload("file", fileName, bytesReader(data))
	if err != nil {
		return nil, err
	}
	return s.Send(ctx, userID, SendReq{ChatID: chatID, Text: text, MediaURL: url})
}

func (s *service) MarkSeen(messageID int64, userID string) error {
	return s.repo.MarkSeen(messageID, userID)
}

func (s *service) ListByChat(chatID int64, limit, offset int) ([]Message, error) {
	return s.repo.ListByChat(chatID, limit, offset)
}

func (s *service) emit(m *Message) error {
	b, _ := json.Marshal(map[string]any{
		"message_id": m.ID, "chat_id": m.ChatID, "user_id": m.UserID,
		"text": m.Text, "media_url": m.MediaURL, "send_time": m.SendTime,
	})
	return s.kafka.Publish(context.Background(), "chat:"+strconv.FormatInt(m.ChatID, 10), b)
}


services/message-service/internal/message/util.go
package message

import (
	"bytes"
	"io"
)

// bytesReader returns an io.Reader for []byte
func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

services/message-service/internal/media/client.go
package media

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
)

type Client struct{ base string }

func New(base string) *Client {
	if base == "" {
		base = "http://media-service:8088"
	}
	return &Client{base: base}
}

func (c *Client) Upload(fieldName, fileName string, r io.Reader) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile(fieldName, fileName)
	_, _ = io.Copy(fw, r)
	_ = w.Close()

	req, _ := http.NewRequest("POST", c.base+"/media/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", io.ErrUnexpectedEOF
	}
	var o struct {
		URL string `json:"url"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&o)
	return o.URL, nil
}


services/message-service/internal/kafka/writer.go
package kafka

import (
	"context"
	"time"

	k "github.com/segmentio/kafka-go"
)

type Writer struct {
	w *k.Writer
}

func NewWriter(bootstrap, topic string) (*Writer, error) {
	w := &k.Writer{
		Addr:         k.TCP(bootstrap),
		Topic:        topic,
		Balancer:     &k.LeastBytes{},
		BatchTimeout: 50 * time.Millisecond,
		RequiredAcks: k.RequireNone,
		Async:        true,
	}
	return &Writer{w: w}, nil
}

func (w *Writer) Close() error { return w.w.Close() }

func (w *Writer) Publish(ctx context.Context, key string, value []byte) error {
	return w.w.WriteMessages(ctx, k.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	})
}


services/message-service/internal/chat/chat.go
package chat

import "time"

type Chat struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:200" json:"name"`
	OwnerID   string    `gorm:"size:64" json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatUser struct {
	ChatID    int64     `gorm:"primaryKey" json:"chat_id"`
	UserID    string    `gorm:"primaryKey;size:64" json:"user_id"`
	Type      string    `gorm:"size:32" json:"type"` // member/admin/…
	CreatedAt time.Time `json:"created_at"`
}

type CreateReq struct {
	Name    string   `json:"name" validate:"required"`
	Members []string `json:"members"` // optional (besides creator)
}

services/message-service/internal/chat/handler.go
package chat

import (
	"net/http"
	"strconv"

	"message-service/internal/shared/httpx"
	"message-service/internal/shared/validate"
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
	c, err := h.svc.Create(uid, in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, c, http.StatusCreated)
	return nil
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	c, err := h.svc.GetByID(id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, c, http.StatusOK)
	return nil
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := qint(r, "limit", 50)
	offset := qint(r, "offset", 0)
	out, err := h.svc.ListMine(uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": out, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) Join(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	if cid == 0 {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Join(cid, uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, ok(), http.StatusOK)
	return nil
}

func (h *Handler) AddUser(w http.ResponseWriter, r *http.Request) error {
	_, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	uid := r.PathValue("user_id")
	if cid == 0 || uid == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.AddUser(cid, uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, ok(), http.StatusOK)
	return nil
}

func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	if cid == 0 {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Leave(cid, uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, ok(), http.StatusOK)
	return nil
}

func (h *Handler) Popular(w http.ResponseWriter, r *http.Request) error {
	top := qint(r, "top", 10)
	ids, err := h.svc.TopPopular(r.Context(), int64(top))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"chat_ids": ids}, http.StatusOK)
	return nil
}

func qint(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		return def
	}
	return n
}
func ok() map[string]string { return map[string]string{"status": "ok"} }

services/message-service/internal/chat/repository.go
package chat

import (
	"message-service/internal/shared/db"

	"gorm.io/gorm"
)

type Repository interface {
	Create(owner string, name string, extra []string) (*Chat, error)
	GetByID(chatID int64) (*Chat, error)
	AddUser(chatID int64, userID, typ string) error
	RemoveUser(chatID int64, userID string) error
	ListByUser(userID string, limit, offset int) ([]Chat, error)
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(owner, name string, extra []string) (*Chat, error) {
	c := &Chat{Name: name, OwnerID: owner}
	if err := r.store.Base.Create(c).Error; err != nil {
		return nil, err
	}
	members := append([]string{owner}, extra...)
	for _, m := range members {
		_ = r.store.Base.FirstOrCreate(&ChatUser{ChatID: c.ID, UserID: m, Type: "member"}).Error
	}
	return c, nil
}

func (r *repo) GetByID(chatID int64) (*Chat, error) {
	var c Chat
	if err := r.store.Base.First(&c, "id = ?", chatID).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repo) AddUser(chatID int64, userID, typ string) error {
	return r.store.Base.FirstOrCreate(&ChatUser{ChatID: chatID, UserID: userID, Type: typ}).Error
}

func (r *repo) RemoveUser(chatID int64, userID string) error {
	return r.store.Base.Delete(&ChatUser{}, "chat_id=? AND user_id=?", chatID, userID).Error
}

func (r *repo) ListByUser(userID string, limit, offset int) ([]Chat, error) {
	var out []Chat
	err := r.store.Base.
		Joins("JOIN chat_users cu ON cu.chat_id = chats.id AND cu.user_id = ?", userID).
		Order("chats.created_at DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

var _ = gorm.ErrRecordNotFound

services/message-service/internal/chat/service.go

package chat

import (
	"context"
	"message-service/internal/redisx"
)

type Service interface {
	Create(owner string, in CreateReq) (*Chat, error)
	GetByID(chatID int64) (*Chat, error)
	AddUser(chatID int64, userID string) error
	Join(chatID int64, userID string) error
	Leave(chatID int64, userID string) error
	ListMine(userID string, limit, offset int) ([]Chat, error)
	IncPopular(ctx context.Context, chatID int64)
	TopPopular(ctx context.Context, n int64) ([]int64, error)
}

type service struct {
	repo Repository
	rds  *redisx.Client
}

func NewService(r Repository, rds *redisx.Client) Service {
	return &service{repo: r, rds: rds}
}

func (s *service) Create(owner string, in CreateReq) (*Chat, error) {
	return s.repo.Create(owner, in.Name, in.Members)
}
func (s *service) GetByID(chatID int64) (*Chat, error) { return s.repo.GetByID(chatID) }
func (s *service) AddUser(chatID int64, userID string) error {
	return s.repo.AddUser(chatID, userID, "member")
}
func (s *service) Join(chatID int64, userID string) error {
	return s.repo.AddUser(chatID, userID, "member")
}
func (s *service) Leave(chatID int64, userID string) error { return s.repo.RemoveUser(chatID, userID) }
func (s *service) ListMine(userID string, limit, offset int) ([]Chat, error) {
	return s.repo.ListByUser(userID, limit, offset)
}
func (s *service) IncPopular(ctx context.Context, chatID int64) { s.rds.IncPopular(ctx, chatID) }
func (s *service) TopPopular(ctx context.Context, n int64) ([]int64, error) {
	return s.rds.TopPopular(ctx, n)
}
