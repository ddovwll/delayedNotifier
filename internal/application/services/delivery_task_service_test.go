package services

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"delayedNotifier/internal/domain/models"

	"github.com/google/uuid"
)

type stubMessageQueueProducer struct {
	publishFunc func(string, []byte, time.Duration) error
	calls       int
	lastKey     string
	lastPayload []byte
	lastDelay   time.Duration
}

func (s *stubMessageQueueProducer) Publish(routingKey string, message []byte, delay time.Duration) error {
	s.calls++
	s.lastKey = routingKey
	s.lastPayload = message
	s.lastDelay = delay
	if s.publishFunc != nil {
		return s.publishFunc(routingKey, message, delay)
	}
	return nil
}

func TestDeliveryTaskService_PublishTask(t *testing.T) {
	producer := &stubMessageQueueProducer{}
	svc := NewDeliveryTaskService(producer, "notifications")

	deliveryTime := time.Now().Add(90 * time.Second).Round(time.Millisecond)
	task := models.DeliveryTask{
		NotificationID: uuid.New(),
		DeliveryTime:   deliveryTime,
	}

	if err := svc.PublishTask(task); err != nil {
		t.Fatalf("PublishTask returned error: %v", err)
	}

	if producer.calls != 1 {
		t.Fatalf("expected Publish to be called once, got %d", producer.calls)
	}

	if producer.lastKey != "notifications" {
		t.Fatalf("expected routing key 'notifications', got %q", producer.lastKey)
	}

	var published models.DeliveryTask
	if err := json.Unmarshal(producer.lastPayload, &published); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if published.NotificationID != task.NotificationID {
		t.Fatalf("expected NotificationID %v, got %v", task.NotificationID, published.NotificationID)
	}

	expectedDelay := time.Until(deliveryTime)
	if diff := expectedDelay - producer.lastDelay; diff < -10*time.Millisecond || diff > 10*time.Millisecond {
		t.Fatalf("unexpected delay: want %v, got %v", expectedDelay, producer.lastDelay)
	}
}

func TestDeliveryTaskService_PublishTaskError(t *testing.T) {
	expectedErr := errors.New("publish error")
	producer := &stubMessageQueueProducer{
		publishFunc: func(string, []byte, time.Duration) error {
			return expectedErr
		},
	}

	svc := NewDeliveryTaskService(producer, "notifications")

	err := svc.PublishTask(models.DeliveryTask{DeliveryTime: time.Now().Add(time.Minute)})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeliveryTaskService_PublishTaskPastTime(t *testing.T) {
	producer := &stubMessageQueueProducer{}
	svc := NewDeliveryTaskService(producer, "notifications")

	err := svc.PublishTask(models.DeliveryTask{DeliveryTime: time.Now().Add(-time.Minute)})
	if !errors.Is(err, ErrDeliveryTimeInPast) {
		t.Fatalf("expected error %v, got %v", ErrDeliveryTimeInPast, err)
	}

	if producer.calls != 0 {
		t.Fatalf("producer should not be called when delivery time is invalid")
	}
}
