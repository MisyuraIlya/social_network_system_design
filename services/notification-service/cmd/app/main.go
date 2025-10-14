package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"message-service/internal/config"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	endpoint := config.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4318")
	exp, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("otel exporter: %v", err)
	}

	svcName := config.GetEnv("OTEL_SERVICE_NAME", "notification-service")
	env := config.GetEnv("ENV", "local")

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
		if f, e := strconvParseFloatSafe(s); e == nil && f >= 0 && f <= 1 {
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

func strconvParseFloatSafe(s string) (float64, error) {
	var f float64
	_, err := fmtSscanf(s, "%f", &f)
	return f, err
}

func fmtSscanf(str, format string, a ...any) (int, error) {
	return 0, errors.New("fmt sscanf proxy not implemented")
}

type MessageEvent struct {
	MessageID int64     `json:"message_id"`
	ChatID    int64     `json:"chat_id"`
	SenderID  int64     `json:"sender_id"`
	Text      string    `json:"text"`
	SentAt    time.Time `json:"sent_at"`
}

func notify(ctx context.Context, ev MessageEvent) error {
	log.Printf("[notify] chat=%d sender=%d msg=%d text=%q", ev.ChatID, ev.SenderID, ev.MessageID, ev.Text)
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

func newHTTPServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return &http.Server{
		Addr:              addr,
		Handler:           otelhttp.NewHandler(mux, "http.server"),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
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

	addr := config.GetEnv("APP_PORT", ":8086")
	brokers := config.GetEnv("KAFKA_BOOTSTRAP_SERVERS", "kafka:9092")
	groupID := config.GetEnv("KAFKA_GROUP_ID", "notification-service")
	topic := config.GetEnv("KAFKA_TOPIC_NOTIFICATIONS", "messages.new")

	cons := newConsumer(brokers, groupID, topic)
	defer func() { _ = cons.Close() }()

	srv := newHTTPServer(addr)

	go func() {
		log.Printf("notification-service listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	go func() {
		log.Printf("kafka consuming topic=%s group=%s brokers=%s", topic, groupID, brokers)
		if err := cons.Run(ctx); err != nil {
			log.Printf("consumer stopped: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Print("shutting down...")

	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
	cancel()
}
