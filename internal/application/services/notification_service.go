package services

import (
	"context"
	"delayedNotifier/internal/application/contracts"
	domaincontracts "delayedNotifier/internal/domain/contracts"
	"delayedNotifier/internal/domain/models"
	"time"

	"github.com/google/uuid"
)

type NotificationConfig struct {
	cacheExpiration time.Duration
}

type NotificationService struct {
	notificationRepository domaincontracts.NotificationRepository
	deliveryTaskService    DeliveryTaskService
	notifierFactory        contracts.NotifierFactory
	retryer                contracts.Retryer
}

func NewNotificationService(
	notificationRepository domaincontracts.NotificationRepository,
	deliveryTaskService DeliveryTaskService,
	notifierFactory contracts.NotifierFactory,
	retryer contracts.Retryer,
) *NotificationService {
	return &NotificationService{
		notificationRepository: notificationRepository,
		deliveryTaskService:    deliveryTaskService,
		notifierFactory:        notifierFactory,
		retryer:                retryer,
	}
}

func (s *NotificationService) Create(ctx context.Context, notification models.Notification) error {
	err := s.notificationRepository.Create(ctx, &notification)
	if err != nil {
		return err
	}

	return s.createDeliveryTask(notification)
}

func (s *NotificationService) createDeliveryTask(notification models.Notification) error {
	task := models.DeliveryTask{
		NotificationID: notification.ID,
		ExecuteAt:      notification.ScheduledAt,
		// todo Retries count
		RetryCount: 5,
		Retries:    0,
		// todo retry time
		NextRetryAt: notification.ScheduledAt.Add(30 * time.Second),
	}

	return s.deliveryTaskService.PublishTask(task)
}

func (s *NotificationService) GetStatus(ctx context.Context, notificationId uuid.UUID) (string, error) {
	notification, err := s.notificationRepository.GetByID(ctx, notificationId)
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
	return s.notificationRepository.Update(ctx, notification)
}

func (s *NotificationService) Notify(ctx context.Context, task models.DeliveryTask) error {
	notification, err := s.notificationRepository.GetByID(ctx, task.NotificationID)
	if err != nil {
		return err
	}

	notifier, err := s.notifierFactory.GetNotifier(notification.Channel)
	if err != nil {
		return err
	}

	return s.retryer.Retry(func() error {
		return notifier.Notify(notification.Recipient, notification.Message)
	})
}
