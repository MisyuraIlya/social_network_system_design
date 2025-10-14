package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"post-service/internal/comment"
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

	commentRepo := comment.NewRepository(store)
	commentSvc := comment.NewService(commentRepo)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	ph := post.NewHandler(postSvc)
	mux.Handle("GET /posts/{post_id}", httpx.Wrap(ph.GetByID))
	mux.Handle("GET /users/{user_id}/posts", httpx.Wrap(ph.ListByUser))

	ch := comment.NewHandler(commentSvc)
	mux.Handle("GET /posts/{post_id}/comments", httpx.Wrap(ch.ListByPost))

	protect := func(pattern string, h http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(h))
	}

	protect("POST /posts", httpx.Wrap(ph.Create))
	protect("POST /posts/{post_id}/like", httpx.Wrap(ph.Like))
	protect("POST /posts/{post_id}/view", httpx.Wrap(ph.AddView))
	protect("POST /posts/upload", httpx.Wrap(ph.UploadAndCreate))

	protect("POST /comments", httpx.Wrap(ch.Create))
	protect("POST /comments/{comment_id}/like", httpx.Wrap(ch.Like))

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
