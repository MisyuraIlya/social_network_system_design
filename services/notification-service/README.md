# Project code dump

- Generated: 2025-10-17 11:38:02+0300
- Root: `/home/spetsar/projects/social_network_system_design/services/notification-service`

cmd/app/main.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"notification-service/internal/notification"
	"notification-service/internal/shared/httpx"
	"notification-service/internal/shared/redisx"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func initOTEL(ctx context.Context) func(context.Context) error {
	endpoint := envOr("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4318")
	exp, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("otel exporter: %v", err)
	}

	svcName := envOr("OTEL_SERVICE_NAME", "notification-service")
	env := envOr("ENV", "local")

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(svcName),
			attribute.String("deployment.environment", env),
		),
	)

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
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	)
	return tp.Shutdown
}

type MessageEvent struct {
	MessageID int64     `json:"message_id"`
	ChatID    int64     `json:"chat_id"`
	UserID    string    `json:"user_id"`
	Text      string    `json:"text"`
	MediaURL  string    `json:"media_url"`
	SendTime  time.Time `json:"send_time"`
}

func notify(ctx context.Context, ev MessageEvent) error {
	log.Printf("[notify] chat=%d sender=%s msg=%d text=%q", ev.ChatID, ev.UserID, ev.MessageID, ev.Text)
	return nil
}

type consumer struct {
	reader *kafka.Reader
}

func newConsumer(brokers, groupID, topic string) *consumer {
	return &consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        strings.Split(brokers, ","),
			GroupID:        groupID,
			Topic:          topic,
			MinBytes:       1,
			MaxBytes:       10 << 20,
			StartOffset:    kafka.FirstOffset,
			CommitInterval: time.Second,
		}),
	}
}

func (c *consumer) Close() error { return c.reader.Close() }

func (c *consumer) Run(ctx context.Context) error {
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			log.Printf("kafka fetch: %v", err)
			continue
		}

		var ev MessageEvent
		if err := json.Unmarshal(m.Value, &ev); err != nil {
			log.Printf("kafka decode: %v (key=%s)", err, string(m.Key))
			_ = c.reader.CommitMessages(ctx, m)
			continue
		}

		if err := notify(ctx, ev); err != nil {
			log.Printf("notify error: %v", err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("kafka commit: %v", err)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown := initOTEL(ctx)
	defer func() {
		c, cc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cc()
		_ = shutdown(c)
	}()

	// Dependencies
	rdb := redisx.OpenFromEnv()
	defer func() { _ = rdb.Close() }()

	repo := notification.NewRedisRepository(rdb)
	svc := notification.NewService(repo)
	h := notification.NewHandler(svc)

	// HTTP Router
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	protect := func(pattern string, handler http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(handler))
	}
	protect("GET /notifications", httpx.Wrap(h.List))
	protect("GET /users/{user_id}/notifications", httpx.Wrap(h.List))
	protect("POST /notifications/{id}/read", httpx.Wrap(h.MarkRead))
	protect("POST /notifications/test", httpx.Wrap(h.CreateTest))

	// Server
	addr := envOr("APP_PORT", ":8086")
	srv := &http.Server{
		Addr:              addr,
		Handler:           otelhttp.NewHandler(mux, "http.server"),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	// Kafka
	brokers := envOr("KAFKA_BOOTSTRAP_SERVERS", "kafka:9092")
	groupID := envOr("KAFKA_GROUP_ID", "notification-service")
	topic := envOr("KAFKA_TOPIC_NOTIFICATIONS", "messages.created")
	cons := newConsumer(brokers, groupID, topic)
	defer func() { _ = cons.Close() }()

	// Start HTTP
	go func() {
		log.Printf("notification-service listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	// Start Kafka consumer
	go func() {
		log.Printf("kafka consuming topic=%s group=%s brokers=%s", topic, groupID, brokers)
		if err := cons.Run(ctx); err != nil {
			log.Printf("consumer stopped: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Print("shutting down...")

	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
	cancel()
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

internal/kafka/consumer.go
package kafka

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type Handler func(ctx context.Context, topic string, key, value []byte) error

type Consumer struct {
	reader *kafka.Reader
	handle Handler
}

func NewConsumer(brokers, groupID, topic string, h Handler) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        strings.Split(brokers, ","),
			GroupID:        groupID,
			Topic:          topic,
			MinBytes:       10e3,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
		}),
		handle: h,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	defer func() {
		_ = c.reader.Close()
	}()

	log.Printf("[Kafka] Consumer started | group=%s | topic=%s | brokers=%v",
		c.reader.Config().GroupID, c.reader.Config().Topic, c.reader.Config().Brokers)

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("[Kafka] Consumer shutting down...")
				return nil
			}
			log.Printf("[Kafka] Fetch error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if c.handle != nil {
			if e := c.handle(ctx, m.Topic, m.Key, m.Value); e != nil {
				log.Printf("[Kafka] Handler error: %v", e)
			}
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("[Kafka] Commit error: %v", err)
		}
	}
}

internal/notification/handler.go
package notification

import (
	"encoding/json"
	"net/http"
	"strconv"

	"notification-service/internal/shared/httpx"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return errUnauthorized("auth required")
	}

	if pathUID := r.PathValue("user_id"); pathUID != "" && pathUID != uid {
		return errUnauthorized("forbidden: cannot read other users' notifications")
	}

	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	items, err := h.svc.List(r.Context(), uid, limit)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"notifications": items}, http.StatusOK)
	return nil
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return errUnauthorized("auth required")
	}
	id := r.PathValue("id")
	if id == "" {
		return errBadReq("missing id")
	}
	if err := h.svc.MarkRead(r.Context(), uid, id); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) CreateTest(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		UserID string         `json:"user_id"`
		Title  string         `json:"title"`
		Body   string         `json:"body"`
		Kind   Kind           `json:"kind"`
		Meta   map[string]any `json:"meta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errBadReq("bad json")
	}
	if req.Kind == "" {
		req.Kind = KindMessage
	}
	n, err := h.svc.Create(r.Context(), req.UserID, req.Kind, req.Title, req.Body, req.Meta)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, n, http.StatusCreated)
	return nil
}

type httpErr struct {
	msg  string
	code int
}

func (e httpErr) Error() string      { return e.msg }
func errBadReq(m string) error       { return httpErr{m, http.StatusBadRequest} }
func errUnauthorized(m string) error { return httpErr{m, http.StatusUnauthorized} }

internal/notification/model.go
package notification

import "time"

type Kind string

const (
	KindMessage Kind = "message"
	KindPost    Kind = "post"
)

type Notification struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Kind      Kind           `json:"kind"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Meta      map[string]any `json:"meta,omitempty"`
	Read      bool           `json:"read"`
	CreatedAt time.Time      `json:"created_at"`
}

internal/notification/repository.go
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
	k := key(userID)
	// Fetch entire list (bounded by reasonable size) and find the index
	vals, err := r.rdb.LRange(ctx, k, 0, 999).Result()
	if err != nil {
		return err
	}
	idx := -1
	var updated string
	for i, v := range vals {
		var n Notification
		if json.Unmarshal([]byte(v), &n) == nil {
			if n.ID == notifID {
				n.Read = true
				b, _ := json.Marshal(n)
				updated = string(b)
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		// nothing to update
		return nil
	}
	// LSET updates in-place; maintains order and is O(1)
	return r.rdb.LSet(ctx, k, int64(idx), updated).Err()
}

internal/notification/service.go
package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, userID string, kind Kind, title, body string, meta map[string]any) (Notification, error)
	List(ctx context.Context, userID string, limit int64) ([]Notification, error)
	MarkRead(ctx context.Context, userID, notifID string) error
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Create(ctx context.Context, userID string, kind Kind, title, body string, meta map[string]any) (Notification, error) {
	n := Notification{
		ID:        uuid.NewString(),
		UserID:    userID,
		Kind:      kind,
		Title:     title,
		Body:      body,
		Meta:      meta,
		CreatedAt: time.Now().UTC(),
	}
	return n, s.repo.Push(ctx, n)
}

func (s *service) List(ctx context.Context, userID string, limit int64) ([]Notification, error) {
	return s.repo.List(ctx, userID, limit)
}

func (s *service) MarkRead(ctx context.Context, userID, notifID string) error {
	return s.repo.MarkRead(ctx, userID, notifID)
}

internal/shared/httpx/httpx.go
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	jw "github.com/golang-jwt/jwt/v5"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) error

type APIError struct {
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
	Status int    `json:"status"`
}

type ctxKey string

const userKey ctxKey = "user_id"

var ErrUnauthorized = errors.New("unauthorized")

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

func secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("replace-this-with-a-strong-secret")
}

func parseJWT(tok string) (string, error) {
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
	if exp, ok := mc["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return "", errors.New("token expired")
	}
	return uid, nil
}

func Wrap(fn HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			WriteError(w, http.StatusBadRequest, err, "")
		}
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	secret := os.Getenv("JWT_SECRET")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret == "" {
			next.ServeHTTP(w, r)
			return
		}
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "missing_bearer")
			return
		}
		token := strings.TrimSpace(h[7:])
		uid, err := parseJWT(token)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid_token")
			return
		}
		ctx := context.WithValue(r.Context(), userKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromCtx(r *http.Request) (string, error) {
	uid, _ := r.Context().Value(userKey).(string)
	if uid == "" {
		return "", ErrUnauthorized
	}
	return uid, nil
}

func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func NowUTC() time.Time { return time.Now().UTC() }

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
	// dev fallback; replace in prod
	return []byte("replace-this-with-a-strong-secret")
}

// Parse validates HS256 JWT and returns the user id from the "sub" claim.
func Parse(tok string) (string, error) {
	t, err := jw.Parse(tok, func(t *jw.Token) (any, error) {
		return secret(), nil
	})
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
	// Optional but recommended: honor exp if present
	if exp, ok := mc["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return "", errors.New("token expired")
	}
	return uid, nil
}

internal/shared/redisx/redisx.go
package redisx

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func OpenFromEnv() *redis.Client {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "redis-message"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	_ = rdb.Ping(context.Background()).Err()
	return rdb
}

