package services

import (
	"context"
	"delayedNotifier/internal/application/contracts"
	domaincontracts "delayedNotifier/internal/domain/contracts"
	"delayedNotifier/internal/domain/models"
	"encoding/json"
	"errors"
	"log"
	"strings"
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

var (
	ErrNotificationRecipientRequired = errors.New("notification recipient is required")
	ErrNotificationMessageRequired   = errors.New("notification message is required")
	ErrNotificationInvalidChannel    = errors.New("notification channel is invalid")
	ErrNotificationScheduledInPast   = errors.New("notification scheduled time must be in the future")
	ErrNotificationAlreadyProcessed  = errors.New("notification already processed")
)

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

func validateNotification(notification *models.Notification) error {
	if notification == nil {
		return errors.New("notification is nil")
	}

	if strings.TrimSpace(notification.Recipient) == "" {
		return ErrNotificationRecipientRequired
	}

	if strings.TrimSpace(notification.Message) == "" {
		return ErrNotificationMessageRequired
	}

	if notification.Channel != models.Telegram && notification.Channel != models.Email {
		return ErrNotificationInvalidChannel
	}

	if notification.ScheduledAt.IsZero() || time.Until(notification.ScheduledAt) <= 0 {
		return ErrNotificationScheduledInPast
	}

	return nil
}

func (s *NotificationService) Create(ctx context.Context, notification *models.Notification) error {
	if err := validateNotification(notification); err != nil {
		return err
	}

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
	if notification.Status == models.Sent || notification.Status == models.Failed {
		return ErrNotificationAlreadyProcessed
	}
	if notification.Status == models.Cancelled {
		return nil
	}

	originalStatus := notification.Status
	updatedAt := time.Now().UTC()
	updated, err := s.notificationRepository.UpdateStatus(ctx, notification.ID, originalStatus, models.Cancelled, updatedAt)
	if err != nil {
		return err
	}

	if !updated {
		latest, getErr := s.notificationRepository.GetByID(ctx, notificationId)
		if getErr != nil {
			return getErr
		}
		notification = latest
		switch notification.Status {
		case models.Sent, models.Failed:
			return ErrNotificationAlreadyProcessed
		case models.Cancelled:
			return nil
		default:
			return errors.New("notification status changed during cancel")
		}
	}

	notification.Status = models.Cancelled
	notification.UpdatedAt = updatedAt

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
		updatedAt := time.Now().UTC()
		originalStatus := notification.Status
		updated, updateErr := s.notificationRepository.UpdateStatus(ctx, notification.ID, originalStatus, models.Failed, updatedAt)
		if updateErr != nil {
			return updateErr
		}

		if !updated {
			latest, getErr := s.notificationRepository.GetByID(ctx, notification.ID)
			if getErr != nil {
				return getErr
			}
			notification = latest
		} else {
			notification.Status = models.Failed
			notification.UpdatedAt = updatedAt
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

	updatedAt := time.Now().UTC()
	originalStatus := notification.Status
	updated, updateErr := s.notificationRepository.UpdateStatus(ctx, notification.ID, originalStatus, models.Sent, updatedAt)
	if updateErr != nil {
		return updateErr
	}

	if !updated {
		latest, getErr := s.notificationRepository.GetByID(ctx, notification.ID)
		if getErr != nil {
			return getErr
		}
		notification = latest
		if notification.Status == models.Cancelled {
			return errors.New("notification is cancelled")
		}
	} else {
		notification.Status = models.Sent
		notification.UpdatedAt = updatedAt
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
