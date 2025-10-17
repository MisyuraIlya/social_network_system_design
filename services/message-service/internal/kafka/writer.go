package kafka

import (
	"context"
	"os"
	"strings"
	"time"

	k "github.com/segmentio/kafka-go"
)

type Writer struct {
	w *k.Writer
}

// NewWriter creates a Kafka writer with configurable durability.
//
// Env overrides (optional):
//   - KAFKA_BOOTSTRAP_SERVERS: "host1:9092,host2:9092" (fallback to arg, then "kafka:9092")
//   - KAFKA_REQUIRED_ACKS: "none" | "one" | "all" (default: "one")
//   - KAFKA_ASYNC: "true" | "false" (default: "false")
func NewWriter(bootstrap, topic string) (*Writer, error) {
	if bootstrap == "" {
		bootstrap = os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	}
	if strings.TrimSpace(bootstrap) == "" {
		bootstrap = "kafka:9092"
	}

	acks := strings.ToLower(strings.TrimSpace(os.Getenv("KAFKA_REQUIRED_ACKS")))
	var requiredAcks k.RequiredAcks
	switch acks {
	case "none":
		requiredAcks = k.RequireNone
	case "all":
		requiredAcks = k.RequireAll
	default:
		// safer default: wait for leader ack
		requiredAcks = k.RequireOne
	}

	async := strings.EqualFold(os.Getenv("KAFKA_ASYNC"), "true")

	w := &k.Writer{
		Addr:         k.TCP(bootstrap),
		Topic:        topic,
		Balancer:     &k.LeastBytes{},
		BatchTimeout: 50 * time.Millisecond,
		RequiredAcks: requiredAcks,
		Async:        async,
	}

	return &Writer{w: w}, nil
}

func (w *Writer) Close() error { return w.w.Close() }

func (w *Writer) Publish(ctx context.Context, key string, value []byte) error {
	return w.w.WriteMessages(ctx, k.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	})
}
