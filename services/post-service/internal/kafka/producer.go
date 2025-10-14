package kafka

import (
	"context"
	"encoding/json"
	"fmt"
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

func NewWriter(bootstrapServers, topic string) (Writer, error) {
	addr := "kafka:9092"
	if strings.TrimSpace(bootstrapServers) != "" {
		addr = bootstrapServers
	}
	w := &kgo.Writer{
		Addr:         kgo.TCP(addr),
		Topic:        topic,
		Balancer:     &kgo.LeastBytes{},
		RequiredAcks: kgo.RequireOne,
		Async:        false,
		BatchTimeout: 50 * time.Millisecond,
	}
	return &writer{w: w}, nil
}

func (wr *writer) WriteJSON(ctx context.Context, v any) error {
	b, err := jsonMarshal(v)
	if err != nil {
		return err
	}
	msg := kgo.Message{Value: b}
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
