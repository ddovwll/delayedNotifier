package message_queue

import (
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
)

type RabbitProducer struct {
	publisher     *rabbitmq.Publisher
	retryStrategy retry.Strategy
}

func NewRabbitProducer(publisher *rabbitmq.Publisher, strategy retry.Strategy) *RabbitProducer {
	return &RabbitProducer{
		publisher:     publisher,
		retryStrategy: strategy,
	}
}

func (p *RabbitProducer) Publish(routingKey string, message []byte, delay time.Duration) error {
	opts := rabbitmq.PublishingOptions{
		Headers: amqp091.Table{
			"x-delay": delay.Milliseconds(),
		},
	}
	return p.publisher.PublishWithRetry(message, routingKey, "application/json", p.retryStrategy, opts)
}
