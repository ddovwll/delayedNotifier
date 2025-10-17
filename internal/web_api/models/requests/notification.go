package requests

import (
	"delayedNotifier/internal/domain/models"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateNotificationRequest struct {
	Channel     models.Channel `json:"channel"`
	Recipient   string         `json:"recipient"`
	Message     string         `json:"message"`
	ScheduledAt time.Time      `json:"scheduled_at"`
}

func (r CreateNotificationRequest) Validate() error {
	if strings.TrimSpace(r.Recipient) == "" {
		return errors.New("recipient is required")
	}

	if strings.TrimSpace(r.Message) == "" {
		return errors.New("message is required")
	}

	if r.Channel != models.Telegram && r.Channel != models.Email {
		return errors.New("channel is invalid")
	}

	if r.ScheduledAt.IsZero() {
		return errors.New("scheduled_at is required")
	}

	if time.Until(r.ScheduledAt) <= 0 {
		return errors.New("scheduled_at must be in the future")
	}

	return nil
}

func (r CreateNotificationRequest) MapToModel() models.Notification {
	return models.Notification{
		ID:          uuid.New(),
		Channel:     r.Channel,
		Recipient:   r.Recipient,
		Message:     r.Message,
		ScheduledAt: r.ScheduledAt,
		Status:      models.Scheduled,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

type GetNotificationStatus struct {
	Status string `json:"status"`
}
