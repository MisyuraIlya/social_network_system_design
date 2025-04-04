package message

type Publisher interface {
	PublishNewMessage(payload []byte) error
}

type publisher struct {
	kafka KafkaAdapter
}

type KafkaAdapter interface {
	Publish([]byte) error
}

func NewPublisher(kafka KafkaAdapter) Publisher {
	return &publisher{kafka: kafka}
}

func (p *publisher) PublishNewMessage(payload []byte) error {
	return p.kafka.Publish(payload)
}
