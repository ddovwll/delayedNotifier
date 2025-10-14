package services

import (
	"delayedNotifier/internal/application/contracts"
	"delayedNotifier/internal/domain/models"
	"encoding/json"
	"time"
)

type DeliveryTaskService struct {
	producer   contracts.MessageQueueProducer
	routingKey string
}

func NewDeliveryTaskService(producer contracts.MessageQueueProducer, routingKey string) *DeliveryTaskService {
	return &DeliveryTaskService{
		producer:   producer,
		routingKey: routingKey,
	}
}

func (s *DeliveryTaskService) PublishTask(task models.DeliveryTask) error {
	bytes, err := json.Marshal(task)
	if err != nil {
		return err
	}

	delay := time.Until(task.DeliveryTime)

	return s.producer.Publish(s.routingKey, bytes, delay)
}
