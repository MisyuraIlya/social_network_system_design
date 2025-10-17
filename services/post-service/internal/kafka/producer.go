package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	kgo "github.com/segmentio/kafka-go"
)

type Writer interface {
	WriteJSON(ctx context.Context, v any) error
	Close() error
}

type writer struct {
	w *kgo.Writer
}

// NewWriter creates a Kafka writer with configurable durability.
// Env overrides (optional):
//   - KAFKA_BOOTSTRAP_SERVERS: "host1:9092,host2:9092" (fallback to arg, then "kafka:9092")
//   - KAFKA_REQUIRED_ACKS: "none" | "one" | "all" (default: "one")
//   - KAFKA_ASYNC: "true" | "false" (default: "false")
func NewWriter(bootstrapServers, topic string) (Writer, error) {
	addr := strings.TrimSpace(bootstrapServers)
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv("KAFKA_BOOTSTRAP_SERVERS"))
	}
	if addr == "" {
		addr = "kafka:9092"
	}

	acksStr := strings.ToLower(strings.TrimSpace(os.Getenv("KAFKA_REQUIRED_ACKS")))
	var requiredAcks kgo.RequiredAcks
	switch acksStr {
	case "none":
		requiredAcks = kgo.RequireNone
	case "all":
		requiredAcks = kgo.RequireAll
	default:
		requiredAcks = kgo.RequireOne
	}

	async := strings.EqualFold(os.Getenv("KAFKA_ASYNC"), "true")

	w := &kgo.Writer{
		Addr:         kgo.TCP(addr),
		Topic:        topic,
		Balancer:     &kgo.LeastBytes{},
		RequiredAcks: requiredAcks,
		Async:        async,
		BatchTimeout: 50 * time.Millisecond,
	}
	return &writer{w: w}, nil
}

func (wr *writer) WriteJSON(ctx context.Context, v any) error {
	b, err := jsonMarshal(v)
	if err != nil {
		return err
	}
	msg := kgo.Message{Value: b, Time: time.Now()}
	return wr.w.WriteMessages(ctx, msg)
}

func (wr *writer) Close() error { return wr.w.Close() }

func jsonMarshal(v any) ([]byte, error) {
	switch t := v.(type) {
	case []byte:
		return t, nil
	default:
		return jsonMarshalStd(v)
	}
}

func jsonMarshalStd(v any) ([]byte, error) {
	type json = struct{}
	_ = json{}
	return jsonMarshalImpl(v)
}

func jsonMarshalImpl(v any) ([]byte, error) { return json.Marshal(v) }

var _ = fmt.Sprintf
