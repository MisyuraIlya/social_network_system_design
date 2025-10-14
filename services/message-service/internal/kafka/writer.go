package kafka

import (
	"context"
	"time"

	k "github.com/segmentio/kafka-go"
)

type Writer struct {
	w *k.Writer
}

func NewWriter(bootstrap, topic string) (*Writer, error) {
	w := &k.Writer{
		Addr:         k.TCP(bootstrap),
		Topic:        topic,
		Balancer:     &k.LeastBytes{},
		BatchTimeout: 50 * time.Millisecond,
		RequiredAcks: k.RequireNone,
		Async:        true,
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
