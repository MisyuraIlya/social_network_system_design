package kafka

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type Handler func(ctx context.Context, topic string, key, value []byte) error

type Consumer struct {
	reader *kafka.Reader
	handle Handler
}

func NewConsumer(brokers, groupID, topic string, h Handler) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        strings.Split(brokers, ","),
			GroupID:        groupID,
			Topic:          topic,
			MinBytes:       10e3,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
		}),
		handle: h,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	defer func() {
		_ = c.reader.Close()
	}()

	log.Printf("[Kafka] Consumer started | group=%s | topic=%s | brokers=%v",
		c.reader.Config().GroupID, c.reader.Config().Topic, c.reader.Config().Brokers)

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("[Kafka] Consumer shutting down...")
				return nil
			}
			log.Printf("[Kafka] Fetch error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if c.handle != nil {
			if e := c.handle(ctx, m.Topic, m.Key, m.Value); e != nil {
				log.Printf("[Kafka] Handler error: %v", e)
			}
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("[Kafka] Commit error: %v", err)
		}
	}
}
