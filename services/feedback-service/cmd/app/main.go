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
	ch.WithLikeService(likeSvc)
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
