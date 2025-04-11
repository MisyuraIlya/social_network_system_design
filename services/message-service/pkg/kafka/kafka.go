package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	Writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *KafkaProducer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	return &KafkaProducer{Writer: writer}
}

func (k *KafkaProducer) Publish(message []byte) error {
	err := k.Writer.WriteMessages(
		context.Background(),
		kafka.Message{
			Value: message,
		},
	)
	if err != nil {
		log.Println("Kafka publish error:", err)
	}
	return err
}
