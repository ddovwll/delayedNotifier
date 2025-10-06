package notifier

import (
	"delayedNotifier/internal/application/contracts"
	"delayedNotifier/internal/domain/models"
	"errors"
)

type NotifierFactory struct {
	emailNotifier    *EmailNotifier
	telegramNotifier *TelegramNotifier
}

func NewNotifierFactory() *NotifierFactory {
	return &NotifierFactory{
		//todo setup config
		emailNotifier:    &EmailNotifier{},
		telegramNotifier: &TelegramNotifier{},
	}
}

func (f *NotifierFactory) GetNotifier(channel models.Channel) (contracts.Notifier, error) {
	switch channel {
	case models.Email:
		return f.emailNotifier, nil
	case models.Telegram:
		return f.telegramNotifier, nil
	default:
		return nil, errors.New("invalid channel")
	}
}
