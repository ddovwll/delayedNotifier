package services

import (
	"context"
	"delayedNotifier/internal/application/contracts"
	domaincontracts "delayedNotifier/internal/domain/contracts"
	"delayedNotifier/internal/domain/models"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
)

type NotificationService struct {
	notificationRepository domaincontracts.NotificationRepository
	deliveryTaskService    *DeliveryTaskService
	notifierFactory        contracts.NotifierFactory
	retryer                contracts.Retryer
	cache                  contracts.Cache
}

func NewNotificationService(
	notificationRepository domaincontracts.NotificationRepository,
	deliveryTaskService *DeliveryTaskService,
	notifierFactory contracts.NotifierFactory,
	retryer contracts.Retryer,
	cache contracts.Cache,
) *NotificationService {
	return &NotificationService{
		notificationRepository: notificationRepository,
		deliveryTaskService:    deliveryTaskService,
		notifierFactory:        notifierFactory,
		retryer:                retryer,
		cache:                  cache,
	}
}

func (s *NotificationService) Create(ctx context.Context, notification *models.Notification) error {
	err := s.notificationRepository.Create(ctx, notification)
	if err != nil {
		return err
	}

	return s.createDeliveryTask(*notification)
}

func (s *NotificationService) createDeliveryTask(notification models.Notification) error {
	task := models.DeliveryTask{
		NotificationID: notification.ID,
		DeliveryTime:   notification.ScheduledAt,
	}

	return s.deliveryTaskService.PublishTask(task)
}

func (s *NotificationService) GetStatus(ctx context.Context, notificationId uuid.UUID) (string, error) {
	notificationJson, err := s.cache.Get(ctx, notificationId.String())
	if err != nil {
		notification, err := s.notificationRepository.GetByID(ctx, notificationId)
		if err != nil {
			return "", err
		}

		bytes, err := json.Marshal(notification)
		if err != nil {
			return "", err
		}

		err = s.cache.Set(ctx, notificationId.String(), string(bytes), 2*time.Hour)
		if err != nil {
			log.Printf("Error caching notification: %v", err)
		}

		return notification.Status.String(), nil
	}

	var notification models.Notification
	err = json.Unmarshal([]byte(notificationJson), &notification)
	if err != nil {
		return "", err
	}

	return notification.Status.String(), nil
}

func (s *NotificationService) CancelNotification(ctx context.Context, notificationId uuid.UUID) error {
	notification, err := s.notificationRepository.GetByID(ctx, notificationId)
	if err != nil {
		return err
	}

	notification.Status = models.Cancelled
	err = s.notificationRepository.Update(ctx, notification)
	if err != nil {
		return err
	}

	_, err = s.cache.Get(ctx, notification.ID.String())
	if err == nil {
		bytes, err := json.Marshal(notification)
		if err != nil {
			return err
		}
		err = s.cache.Set(ctx, notification.ID.String(), string(bytes), 2*time.Hour)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *NotificationService) Notify(ctx context.Context, task models.DeliveryTask) error {
	notification, err := s.notificationRepository.GetByID(ctx, task.NotificationID)
	if err != nil {
		return err
	}
	if notification.Status == models.Cancelled {
		return errors.New("notification is cancelled")
	}

	notifier, err := s.notifierFactory.GetNotifier(notification.Channel)
	if err != nil {
		return err
	}

	err = s.retryer.Retry(func() error {
		return notifier.Notify(ctx, notification.Recipient, notification.Message)
	})
	outerErr := err
	if err != nil {
		notification.Status = models.Failed
		err := s.notificationRepository.Update(ctx, notification)
		if err != nil {
			return err
		}

		_, err = s.cache.Get(ctx, notification.ID.String())
		if err == nil {
			bytes, err := json.Marshal(notification)
			if err != nil {
				return err
			}
			err = s.cache.Set(ctx, notification.ID.String(), string(bytes), 2*time.Hour)
			if err != nil {
				return err
			}
		}

		return outerErr
	}

	notification.Status = models.Sent
	err = s.notificationRepository.Update(ctx, notification)
	if err != nil {
		return err
	}

	_, err = s.cache.Get(ctx, notification.ID.String())
	if err == nil {
		bytes, err := json.Marshal(notification)
		if err != nil {
			return err
		}
		err = s.cache.Set(ctx, notification.ID.String(), string(bytes), 2*time.Hour)
		if err != nil {
			return err
		}
	}

	return nil
}
