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
