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

	// Public (optional user_id in path; falls back to auth if missing)
	mux.Handle("GET /users/{user_id}/notifications", httpx.Wrap(h.List))

	// Protected
	protect := func(pattern string, handler http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(handler))
	}
	protect("GET /notifications", httpx.Wrap(h.List))
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
