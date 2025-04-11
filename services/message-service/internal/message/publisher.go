package message

// Publisher defines an interface for publishing events.
type Publisher interface {
	PublishNewMessage(payload []byte) error
}

type publisher struct {
	kafka KafkaAdapter
}

// KafkaAdapter defines the required Kafka operations.
type KafkaAdapter interface {
	Publish([]byte) error
}

// NewPublisher creates a new Publisher using the Kafka adapter.
func NewPublisher(kafka KafkaAdapter) Publisher {
	return &publisher{kafka: kafka}
}

func (p *publisher) PublishNewMessage(payload []byte) error {
	return p.kafka.Publish(payload)
}
