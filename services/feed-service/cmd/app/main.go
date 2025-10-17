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
