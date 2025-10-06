package models

import (
	"time"

	"github.com/google/uuid"
)

type DeliveryTask struct {
	NotificationID uuid.UUID
	ExecuteAt      time.Time
	RetryCount     int
	Retries        int
	NextRetryAt    time.Time
}
