package message_queue

import (
	"context"
	"delayedNotifier/internal/application/services"
	"delayedNotifier/internal/domain/models"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
)

type RabbitConsumer struct {
	consumer            *rabbitmq.Consumer
	retryStrategy       retry.Strategy
	notificationService *services.NotificationService
	channel             *rabbitmq.Channel
}

func NewRabbitConsumer(consumer *rabbitmq.Consumer, service *services.NotificationService, strategy retry.Strategy, channel *rabbitmq.Channel) *RabbitConsumer {
	return &RabbitConsumer{
		consumer:            consumer,
		notificationService: service,
		retryStrategy:       strategy,
		channel:             channel,
	}
}

func (c *RabbitConsumer) StartConsumer(ctx context.Context, workers int) {
	messages := make(chan []byte, 1024)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Go(func() {
			c.worker(ctx, messages)
		})
	}

	consumeErrCh := make(chan error, 1)
	go func() {
		consumeErrCh <- c.consumer.ConsumeWithRetry(messages, c.retryStrategy)
	}()

	var consumeErr error
	var consumeFinished bool

	select {
	case <-ctx.Done():
	case consumeErr = <-consumeErrCh:
		consumeFinished = true
	}

	if err := c.channel.Close(); err != nil {
		log.Printf("Rabbit Consumer Error: %s", err)
	}

	if !consumeFinished {
		consumeErr = <-consumeErrCh
	}

	if consumeErr != nil && ctx.Err() == nil {
		log.Printf("Rabbit Consumer Error: %s", consumeErr)
	}

	close(messages)
	wg.Wait()
}

func (c *RabbitConsumer) worker(parentCtx context.Context, ch <-chan []byte) {
	for {
		select {
		case <-parentCtx.Done():
			return
		case bytes, ok := <-ch:
			if !ok {
				return
			}

			var task models.DeliveryTask
			if err := json.Unmarshal(bytes, &task); err != nil {
				log.Printf("Unmarshal Error: %s", err)
				continue
			}

			ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
			if err := c.notificationService.Notify(ctx, task); err != nil {
				log.Printf("Notify Error: %s", err)
			}
			cancel()
		}
	}
}
