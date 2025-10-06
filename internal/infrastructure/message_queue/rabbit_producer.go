package message_queue

import (
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
)

type RabbitProducer struct {
	publisher     *rabbitmq.Publisher
	retryStrategy retry.Strategy
}

func NewRabbitProducer(publisher *rabbitmq.Publisher) *RabbitProducer {
	return &RabbitProducer{publisher: publisher}
}

// Publish for json messages
func (p *RabbitProducer) Publish(topic string, message []byte) error {
	return p.publisher.PublishWithRetry(message, topic, "application/json", p.retryStrategy)
}
