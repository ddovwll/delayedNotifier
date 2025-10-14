package contracts

import "time"

type MessageQueueProducer interface {
	Publish(routingKey string, message []byte, delay time.Duration) error
}
