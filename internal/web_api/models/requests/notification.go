package requests

import (
	"time"
)

type CreateNotificationRequest struct {
	Channel     int       `json:"channel"`
	Recipient   string    `json:"recipient"`
	Message     string    `json:"message"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

type GetNotificationStatus struct {
	Status string `json:"status"`
}
