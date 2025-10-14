package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"media-service/internal/media"
	"media-service/internal/shared/httpx"
	"media-service/internal/storage/s3"

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

func initOTEL(ctx context.Context) func(context.Context) error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4318"
	}
	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("otel exporter: %v", err)
	}
	res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(os.Getenv("OTEL_SERVICE_NAME")),
		attribute.String("deployment.environment", "local"),
	))
	tp := trace.NewTracerProvider(trace.WithBatcher(exp), trace.WithResource(res))
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

	s3cfg := s3.Config{
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		UseSSL:    false,
		Bucket:    envOr("S3_BUCKET", "media"),
	}
	store, err := s3.New(s3cfg)
	if err != nil {
		log.Fatalf("s3: %v", err)
	}
	if err := store.EnsureBucket(ctx); err != nil {
		log.Fatalf("s3 ensure bucket: %v", err)
	}

	svc := media.NewService(store)
	h := media.NewHandler(svc)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.Handle("GET /media/{key}", otelhttp.NewHandler(http.HandlerFunc(h.RedirectToSignedGet), "media.get"))

	protected := func(pattern string, handler http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(handler))
	}
	protected("POST /media/upload", otelhttp.NewHandler(http.HandlerFunc(h.Upload), "media.upload"))
	protected("DELETE /media/{key}", otelhttp.NewHandler(http.HandlerFunc(h.Delete), "media.delete"))
	protected("POST /media/presign", otelhttp.NewHandler(http.HandlerFunc(h.PresignPut), "media.presign"))

	addr := envOr("APP_PORT", ":8088")
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	log.Printf("media-service listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
