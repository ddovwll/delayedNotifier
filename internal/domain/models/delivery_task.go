package models

import (
	"time"

	"github.com/google/uuid"
)

type DeliveryTask struct {
	NotificationID uuid.UUID `json:"notification_id"`
	DeliveryTime   time.Time `json:"delivery_time"`
}
