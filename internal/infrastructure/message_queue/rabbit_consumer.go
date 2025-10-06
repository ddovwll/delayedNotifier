package message_queue

import (
	"context"
	"delayedNotifier/internal/application/services"
	"delayedNotifier/internal/domain/models"
	"encoding/json"
	"log"
	"sync"

	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
)

type RabbitConsumer struct {
	consumer            *rabbitmq.Consumer
	retryStrategy       retry.Strategy
	notificationService *services.NotificationService
}

func NewRabbitConsumer(consumer *rabbitmq.Consumer, service *services.NotificationService, strategy retry.Strategy) *RabbitConsumer {
	return &RabbitConsumer{
		consumer:            consumer,
		notificationService: service,
		retryStrategy:       strategy,
	}
}

func (c *RabbitConsumer) StartConsumer(ctx context.Context, workers int) {
	messages := make(chan []byte, 1024)
	wg := &sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Go(func() {
			c.worker(ctx, messages)
		})
	}

	err := c.consumer.ConsumeWithRetry(messages, c.retryStrategy)
	if err != nil {
		log.Printf("Rabbit Consumer Error: %s", err)
	}
}

func (c *RabbitConsumer) worker(ctx context.Context, ch <-chan []byte) {
	for bytes := range ch {
		var task models.DeliveryTask
		err := json.Unmarshal(bytes, &task)
		if err != nil {
			log.Printf("Unmarshal Error: %s", err)
		}

		err = c.notificationService.Notify(ctx, task)
		if err != nil {
			log.Printf("Notify Error: %s", err)
		}
	}
}
