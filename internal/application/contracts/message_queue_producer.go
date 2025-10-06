package contracts

type MessageQueueProducer interface {
	Publish(topic string, message []byte) error
}
