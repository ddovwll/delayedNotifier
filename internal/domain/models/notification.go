package models

import (
	"time"

	"github.com/google/uuid"
)

type Status int

const (
	Pending Status = iota
	Scheduled
	Sent
	Failed
	Cancelled
	Retrying
)

func (s Status) String() string {
	switch s {
	case Pending:
		return "Pending"
	case Scheduled:
		return "Scheduled"
	case Sent:
		return "Sent"
	case Failed:
		return "Failed"
	case Cancelled:
		return "Cancelled"
	case Retrying:
		return "Retrying"
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
	ID          uuid.UUID
	Channel     Channel
	Recipient   string
	Message     string
	ScheduledAt time.Time
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
