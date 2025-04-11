package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokerURL, topic string) *Producer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{brokerURL},
		Topic:   topic,
	})

	return &Producer{writer: writer}
}

func (p *Producer) Publish(ctx context.Context, key string, message []byte) error {
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: message,
	})

	if err != nil {
		log.Printf("Kafka publish error: %v", err)
		return err
	}

	log.Println("Published message to Kafka")
	return nil
}

func (p *Producer) Close() {
	p.writer.Close()
}
