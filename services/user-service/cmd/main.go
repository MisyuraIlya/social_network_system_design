package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"users-service/internal/user"
	"users-service/pkg/db"

	// Prometheus metrics endpoint
	"github.com/prometheus/client_golang/prometheus/promhttp"

	// OpenTelemetry core
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	// HTTP middleware instrumentation
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	// GORM tracing plugin
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

type ShardPicker interface {
	Pick(shardID int) *gorm.DB
	ForcePrimary(shardID int) *gorm.DB
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func initTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	endpoint := env("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4318")

	exp, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otlptracehttp: %w", err)
	}

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			attribute.String("service.version", "1.0.0"),
			attribute.String("deployment.environment", env("ENV", "local")),
		),
	)

	ratio := 1.0
	if v := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
			ratio = f
		}
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(ratio))),
		trace.WithBatcher(exp,
			trace.WithMaxExportBatchSize(512),
			trace.WithBatchTimeout(3*time.Second),
		),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Honor W3C TraceContext + Baggage for inbound/outbound requests.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		n = 1
	}
	return n
}

func main() {
	ctx := context.Background()

	shutdown, err := initTracer(ctx, "user-service")
	if err != nil {
		log.Fatalf("otel init failed: %v", err)
	}
	// Give exporter time to flush on stop.
	defer func() {
		c, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = shutdown(c)
	}()

	store := db.OpenFromEnv()

	// Enable SQL spans from GORM.
	if err := store.Base.Use(tracing.NewPlugin()); err != nil {
		log.Fatalf("gorm otel plugin failed: %v", err)
	}

	// Auto-migrate across shards if requested.
	if os.Getenv("AUTO_MIGRATE") == "true" {
		numShards := mustAtoi(os.Getenv("NUM_SHARDS"))
		for i := 0; i < numShards; i++ {
			if err := store.ForcePrimary(i).AutoMigrate(&user.User{}); err != nil {
				log.Fatalf("migration failed on shard %d: %v", i, err)
			}
		}
	}

	repo := user.NewUserRepository(store)
	svc := user.NewUserService(repo)
	handler := user.NewUserHandler(svc)

	// App routes.
	api := http.NewServeMux()
	user.RegisterRoutes(api, handler)

	// Root mux: /metrics + OTel-instrumented app.
	root := http.NewServeMux()
	root.Handle("/metrics", promhttp.Handler())
	root.Handle("/", otelhttp.NewHandler(api, "http.server"))

	addr := env("APP_PORT", ":8081")
	srv := &http.Server{
		Addr:              addr,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	fmt.Printf("User Service listening on %s\n", addr)
	log.Fatal(srv.ListenAndServe())
}
