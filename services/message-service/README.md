# Project code dump

- Generated: 2025-10-18 19:02:53+0300
- Root: `/home/spetsar/projects/social_network_system_design/services/message-service`

cmd/app/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"message-service/internal/chat"
	"message-service/internal/idem"
	"message-service/internal/kafka"
	"message-service/internal/media"
	"message-service/internal/message"
	"message-service/internal/migrate"
	"message-service/internal/ratelimit"
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

	store := db.OpenFromEnv()

	rds := redisx.NewClientFromEnv()

	limiter := ratelimit.New(rds)
	idemStore := idem.New(rds)

	kWriter, err := kafka.NewWriter(os.Getenv("KAFKA_BOOTSTRAP_SERVERS"), "messages.created")
	if err != nil {
		log.Fatalf("kafka writer: %v", err)
	}
	defer kWriter.Close()

	mediaCli := media.New(os.Getenv("MEDIA_SERVICE_URL"))

	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err := migrate.AutoMigrateAll(store); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}

	chatRepo := chat.NewRepository(store)
	chatSvc := chat.NewService(chatRepo, rds)

	msgRepo := message.NewRepository(store)
	msgSvc := message.NewService(msgRepo, chatSvc, rds, kWriter, mediaCli)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	ch := chat.NewHandler(chatSvc)
	mh := message.NewHandler(msgSvc).WithIdem(idemStore)

	mux.Handle("GET /chats/{chat_id}", httpx.Wrap(ch.GetByID))

	protect := func(pattern string, h http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(h))
	}

	sendLimit := func(next http.Handler) http.Handler {
		return limiter.LimitHTTP(20, 10*time.Second, func(r *http.Request) (string, error) {
			return httpx.UserFromCtx(r)
		}, next)
	}
	readLimit := func(next http.Handler) http.Handler {
		return limiter.LimitHTTP(60, 10*time.Second, func(r *http.Request) (string, error) {
			return httpx.UserFromCtx(r)
		}, next)
	}

	// Chats
	protect("POST /chats", httpx.Wrap(ch.Create))
	protect("GET /chats", httpx.Wrap(ch.ListMine))
	protect("POST /chats/{chat_id}/join", httpx.Wrap(ch.Join))
	protect("POST /chats/{chat_id}/add/{user_id}", httpx.Wrap(ch.AddUser))
	protect("POST /chats/{chat_id}/leave", httpx.Wrap(ch.Leave))
	mux.Handle("GET /chats/popular", httpx.Wrap(ch.Popular))

	protect("GET /chats/{chat_id}/messages", readLimit(httpx.Wrap(mh.ListByChat)))
	protect("POST /messages", sendLimit(httpx.Wrap(mh.Send)))
	protect("POST /messages/upload", sendLimit(httpx.Wrap(mh.UploadAndSend)))
	protect("POST /messages/{message_id}/seen", readLimit(httpx.Wrap(mh.MarkSeen)))

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

internal/chat/chat.go
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

internal/chat/handler.go
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
	actorID, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	uid := r.PathValue("user_id")
	if cid == 0 || uid == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.AddUser(cid, actorID, uid); err != nil {
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

internal/chat/repository.go
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
	IsMember(chatID int64, userID string) (bool, error)
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
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

func (r *repo) IsMember(chatID int64, userID string) (bool, error) {
	var n int64
	if err := r.store.Base.
		Model(&ChatUser{}).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

var _ = gorm.ErrRecordNotFound

internal/chat/service.go
package chat

import (
	"context"
	"fmt"
	"message-service/internal/redisx"
)

type Service interface {
	Create(owner string, in CreateReq) (*Chat, error)
	GetByID(chatID int64) (*Chat, error)
	AddUser(chatID int64, actorID string, userID string) error
	Join(chatID int64, userID string) error
	Leave(chatID int64, userID string) error
	ListMine(userID string, limit, offset int) ([]Chat, error)
	IncPopular(ctx context.Context, chatID int64)
	TopPopular(ctx context.Context, n int64) ([]int64, error)

	// NEW:
	IsMember(chatID int64, userID string) (bool, error)
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
func (s *service) GetByID(chatID int64) (*Chat, error) {
	return s.repo.GetByID(chatID)
}

func (s *service) AddUser(chatID int64, actorID string, userID string) error {
	chat, err := s.repo.GetByID(chatID)
	if err != nil {
		return err
	}
	if chat.OwnerID != actorID {
		return fmt.Errorf("forbidden: only owner can add users")
	}
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

func (s *service) IsMember(chatID int64, userID string) (bool, error) {
	return s.repo.IsMember(chatID, userID)
}

internal/idem/idem.go
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

internal/kafka/writer.go
package kafka

import (
	"context"
	"os"
	"strings"
	"time"

	k "github.com/segmentio/kafka-go"
)

type Writer struct {
	w *k.Writer
}

// NewWriter creates a Kafka writer with configurable durability.
//
// Env overrides (optional):
//   - KAFKA_BOOTSTRAP_SERVERS: "host1:9092,host2:9092" (fallback to arg, then "kafka:9092")
//   - KAFKA_REQUIRED_ACKS: "none" | "one" | "all" (default: "one")
//   - KAFKA_ASYNC: "true" | "false" (default: "false")
func NewWriter(bootstrap, topic string) (*Writer, error) {
	if bootstrap == "" {
		bootstrap = os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	}
	if strings.TrimSpace(bootstrap) == "" {
		bootstrap = "kafka:9092"
	}

	acks := strings.ToLower(strings.TrimSpace(os.Getenv("KAFKA_REQUIRED_ACKS")))
	var requiredAcks k.RequiredAcks
	switch acks {
	case "none":
		requiredAcks = k.RequireNone
	case "all":
		requiredAcks = k.RequireAll
	default:
		// safer default: wait for leader ack
		requiredAcks = k.RequireOne
	}

	async := strings.EqualFold(os.Getenv("KAFKA_ASYNC"), "true")

	w := &k.Writer{
		Addr:         k.TCP(bootstrap),
		Topic:        topic,
		Balancer:     &k.LeastBytes{},
		BatchTimeout: 50 * time.Millisecond,
		RequiredAcks: requiredAcks,
		Async:        async,
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

internal/media/client.go
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

func (c *Client) Upload(fieldName, fileName string, r io.Reader, bearer string) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile(fieldName, fileName)
	_, _ = io.Copy(fw, r)
	_ = w.Close()

	req, _ := http.NewRequest("POST", c.base+"/media/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
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

internal/message/handler.go
package message

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"message-service/internal/idem"
	"message-service/internal/shared/httpx"
	"message-service/internal/shared/validate"
)

type Handler struct {
	svc  Service
	idem idem.Store
}

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) WithIdem(s idem.Store) *Handler {
	h.idem = s
	return h
}

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

	if h.idem != nil {
		if key := r.Header.Get("Idempotency-Key"); key != "" {
			ok, e := h.idem.PutNX(r.Context(), "send:"+uid+":"+strconv.FormatInt(in.ChatID, 10)+":"+key, 24*time.Hour)
			if e != nil {
				return e
			}
			if !ok {
				httpx.WriteJSON(w, map[string]any{"error": "duplicate request"}, http.StatusConflict)
				return nil
			}
		}
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
	bearer := httpx.BearerToken(r)

	if h.idem != nil {
		if key := r.Header.Get("Idempotency-Key"); key != "" {
			ok, e := h.idem.PutNX(r.Context(), "send-upload:"+uid+":"+strconv.FormatInt(chatID, 10)+":"+key, 24*time.Hour)
			if e != nil {
				return e
			}
			if !ok {
				httpx.WriteJSON(w, map[string]any{"error": "duplicate request"}, http.StatusConflict)
				return nil
			}
		}
	}

	m, err := h.svc.SendWithUpload(r.Context(), uid, chatID, fh.Filename, data, text, bearer)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, m, http.StatusCreated)
	return nil
}

func (h *Handler) ListByChat(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	limit := qint(r, "limit", 50)
	offset := qint(r, "offset", 0)

	items, err := h.svc.ListByChat(uid, cid, limit, offset)
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

internal/message/message.go
package message

import "time"

type Message struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	UserID        string    `gorm:"size:64" json:"user_id"`
	ChatID        int64     `gorm:"index" json:"chat_id"`
	Text          string    `json:"text"`
	MediaURL      string    `gorm:"size:512" json:"media_url"`
	SendTime      time.Time `json:"send_time"`
	DeliveredTime time.Time `json:"delivered_time"`
}

type SendReq struct {
	ChatID   int64  `json:"chat_id" validate:"required"`
	Text     string `json:"text"`
	MediaURL string `json:"media_url"`
}

type MessageSeen struct {
	MessageID int64     `gorm:"primaryKey;index" json:"message_id"`
	UserID    string    `gorm:"primaryKey;size:64;index" json:"user_id"`
	SeenAt    time.Time `json:"seen_at"`
}

internal/message/repository.go
package message

import (
	"time"

	"message-service/internal/shared/db"

	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(m *Message) (*Message, error)
	MarkSeen(messageID int64, userID string) error
	ListByChat(chatID int64, limit, offset int) ([]Message, error)

	// NEW:
	GetByID(messageID int64) (*Message, error)
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
	ms := &MessageSeen{
		MessageID: messageID,
		UserID:    userID,
		SeenAt:    time.Now(),
	}
	return r.store.Base.Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "message_id"}, {Name: "user_id"}},
			DoNothing: true,
		},
	).Create(ms).Error
}

func (r *repo) ListByChat(chatID int64, limit, offset int) ([]Message, error) {
	var out []Message
	err := r.store.Base.
		Where("chat_id = ?", chatID).
		Order("id DESC").Limit(limit).Offset(offset).
		Find(&out).Error
	return out, err
}

func (r *repo) GetByID(messageID int64) (*Message, error) {
	var m Message
	if err := r.store.Base.First(&m, "id = ?", messageID).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

internal/message/service.go
package message

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"message-service/internal/chat"
	"message-service/internal/kafka"
	"message-service/internal/media"
	"message-service/internal/redisx"
)

type Service interface {
	Send(ctx context.Context, userID string, in SendReq) (*Message, error)
	SendWithUpload(ctx context.Context, userID string, chatID int64, fileName string, fileData []byte, text string, bearer string) (*Message, error)
	MarkSeen(messageID int64, userID string) error
	ListByChat(userID string, chatID int64, limit, offset int) ([]Message, error)
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

var errForbidden = errors.New("forbidden") // simple sentinel

func (s *service) Send(ctx context.Context, userID string, in SendReq) (*Message, error) {
	if _, err := s.chats.GetByID(in.ChatID); err != nil {
		return nil, err
	}
	if ok, err := s.chats.IsMember(in.ChatID, userID); err != nil {
		return nil, err
	} else if !ok {
		return nil, errForbidden
	}

	m := &Message{
		UserID:   userID,
		ChatID:   in.ChatID,
		Text:     in.Text,
		MediaURL: in.MediaURL,
		SendTime: time.Now(),
	}
	res, err := s.repo.Create(m)
	if err != nil {
		return nil, err
	}

	s.chats.IncPopular(ctx, in.ChatID)
	_ = s.emit(res)

	return res, nil
}

func (s *service) SendWithUpload(ctx context.Context, userID string, chatID int64, fileName string, data []byte, text string, bearer string) (*Message, error) {
	// ensure chat exists
	if _, err := s.chats.GetByID(chatID); err != nil {
		return nil, err
	}
	if ok, err := s.chats.IsMember(chatID, userID); err != nil {
		return nil, err
	} else if !ok {
		return nil, errForbidden
	}

	url, err := s.media.Upload("file", fileName, bytesReader(data), bearer)
	if err != nil {
		return nil, err
	}
	return s.Send(ctx, userID, SendReq{ChatID: chatID, Text: text, MediaURL: url})
}

func (s *service) MarkSeen(messageID int64, userID string) error {
	m, err := s.repo.GetByID(messageID)
	if err != nil {
		return err
	}
	if ok, err := s.chats.IsMember(m.ChatID, userID); err != nil {
		return err
	} else if !ok {
		return errForbidden
	}
	return s.repo.MarkSeen(messageID, userID)
}

func (s *service) ListByChat(userID string, chatID int64, limit, offset int) ([]Message, error) {
	if ok, err := s.chats.IsMember(chatID, userID); err != nil {
		return nil, err
	} else if !ok {
		return nil, errForbidden
	}
	return s.repo.ListByChat(chatID, limit, offset)
}

func (s *service) emit(m *Message) error {
	b, _ := json.Marshal(map[string]any{
		"message_id": m.ID, "chat_id": m.ChatID, "user_id": m.UserID,
		"text": m.Text, "media_url": m.MediaURL, "send_time": m.SendTime,
	})
	return s.kafka.Publish(context.Background(), "chat:"+strconv.FormatInt(m.ChatID, 10), b)
}

internal/message/util.go
package message

import (
	"bytes"
	"io"
)

// bytesReader returns an io.Reader for []byte
func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

internal/migrate/migrate.go
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
		&message.MessageSeen{},
	)
}

internal/ratelimit/ratelimit.go
package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"message-service/internal/redisx"
	"message-service/internal/shared/httpx"
)

type Limiter struct {
	R *redisx.Client
}

func New(r *redisx.Client) *Limiter { return &Limiter{R: r} }

func (l *Limiter) AllowSliding(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, error) {
	k := "rl:" + key
	pipe := l.R.R.TxPipeline()
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

internal/redisx/cache.go
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

internal/shared/db/single.go
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
	user := def(os.Getenv("DB_USER"), "message")
	pass := def(os.Getenv("DB_PASSWORD"), "messagepass")
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

internal/shared/httpx/httpx.go
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
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "missing_bearer")
			return
		}
		tok := strings.TrimSpace(h[7:])
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

internal/shared/validate/validate.go
package validate

import "github.com/go-playground/validator/v10"

var v = validator.New()

func Struct(s any) error { return v.Struct(s) }

