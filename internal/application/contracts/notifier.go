package contracts

import (
	"context"
	"delayedNotifier/internal/domain/models"
)

type Notifier interface {
	Notify(ctx context.Context, recipient, message string) error
}

type NotifierFactory interface {
	GetNotifier(channel models.Channel) (Notifier, error)
}
