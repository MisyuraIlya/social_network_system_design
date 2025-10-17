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
