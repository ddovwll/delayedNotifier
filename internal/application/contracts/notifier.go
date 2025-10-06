package contracts

import "delayedNotifier/internal/domain/models"

type Notifier interface {
	Notify(recipient, message string) error
}

type NotifierFactory interface {
	GetNotifier(channel models.Channel) (Notifier, error)
}
