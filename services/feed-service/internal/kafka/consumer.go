package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"feed-service/internal/feed"

	kf "github.com/segmentio/kafka-go"
)

type PostHandler func(ctx context.Context, ev feed.PostEvent) error

func StartConsumer(ctx context.Context, bootstrap, topic, groupID string, handle PostHandler) error {
	r := kf.NewReader(kf.ReaderConfig{
		Brokers:  strings.Split(bootstrap, ","),
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
		MaxWait:  2 * time.Second,
	})
	defer r.Close()

	log.Printf("kafka consumer started group=%s topic=%s", groupID, topic)

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			return err
		}
		var ev feed.PostEvent
		if err := json.Unmarshal(m.Value, &ev); err != nil {
			log.Printf("kafka: bad payload: %v", err)
			continue
		}
		if err := handle(ctx, ev); err != nil {
			log.Printf("handle post event: %v", err)
		}
	}
}
