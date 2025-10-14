package requests

import (
	"delayedNotifier/internal/domain/models"
	"time"

	"github.com/google/uuid"
)

type CreateNotificationRequest struct {
	Channel     models.Channel `json:"channel"`
	Recipient   string         `json:"recipient"`
	Message     string         `json:"message"`
	ScheduledAt time.Time      `json:"scheduled_at"`
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
