package kafka

import (
	"context"
	"log"
	"strings"
	"time"

	"feed-service/configs"

	"github.com/segmentio/kafka-go"
)

// Producer provides a way to publish messages to Kafka.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer using the provided config.
func NewProducer(cfg *configs.Config) *Producer {
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        cfg.KafkaTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll, // or kafka.RequireOne, etc.
	}
	return &Producer{writer: w}
}

// PublishMessage sends a message to the configured Kafka topic.
func (p *Producer) PublishMessage(ctx context.Context, key, value []byte) error {
	msg := kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}
	return p.writer.WriteMessages(ctx, msg)
}

// Close closes the underlying writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Consumer listens for messages from Kafka.
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer creates a new Kafka consumer.
func NewConsumer(cfg *configs.Config) *Consumer {
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  cfg.KafkaGroupID,
		Topic:    cfg.KafkaTopic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	return &Consumer{reader: r}
}

// StartListening continuously reads messages and processes them.
func (c *Consumer) StartListening(ctx context.Context, handleFunc func(kafka.Message)) {
	log.Println("Kafka consumer started...")
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		handleFunc(m)
	}
}

// Close closes the Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
