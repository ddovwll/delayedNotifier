package services

import (
	"delayedNotifier/internal/application/contracts"
	"delayedNotifier/internal/domain/models"
	"encoding/json"
)

type DeliveryTaskService struct {
	producer contracts.MessageQueueProducer
	topic    string
}

func NewDeliveryTaskService(producer contracts.MessageQueueProducer, topic string) *DeliveryTaskService {
	return &DeliveryTaskService{
		producer: producer,
		topic:    topic,
	}
}

func (s *DeliveryTaskService) PublishTask(task models.DeliveryTask) error {
	bytes, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return s.producer.Publish(s.topic, bytes)
}
