package models

import (
	"time"

	"github.com/google/uuid"
)

type Status int

const (
	Scheduled Status = iota
	Sent
	Failed
	Cancelled
)

func (s Status) String() string {
	switch s {
	case Scheduled:
		return "Scheduled"
	case Sent:
		return "Sent"
	case Failed:
		return "Failed"
	case Cancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

type Channel int

const (
	Telegram Channel = iota
	Email
)

type Notification struct {
	ID          uuid.UUID `json:"id"`
	Channel     Channel   `json:"channel"`
	Recipient   string    `json:"recipient"`
	Message     string    `json:"message"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
