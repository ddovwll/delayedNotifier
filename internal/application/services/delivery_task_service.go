package services

import (
	"delayedNotifier/internal/application/contracts"
	"delayedNotifier/internal/domain/models"
	"encoding/json"
	"errors"
	"time"
)

var ErrDeliveryTimeInPast = errors.New("delivery time must be in the future")

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
	delay := time.Until(task.DeliveryTime)
	if delay <= 0 {
		return ErrDeliveryTimeInPast
	}

	bytes, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return s.producer.Publish(s.routingKey, bytes, delay)
}
