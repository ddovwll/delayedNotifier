package notifier

import (
	"delayedNotifier/internal/application/contracts"
	"delayedNotifier/internal/domain/models"
	"delayedNotifier/internal/infrastructure/notifier/notifier_telegram"
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type NotifierFactory struct {
	emailNotifier    *EmailNotifier
	telegramNotifier *notifier_telegram.TelegramNotifier
}

func NewNotifierFactory(bot *tgbotapi.BotAPI, repository *notifier_telegram.TelegramNotifierRepository) *NotifierFactory {
	tg := notifier_telegram.NewTelegramNotifier(bot, repository)

	return &NotifierFactory{
		emailNotifier:    NewEmailNotifier(),
		telegramNotifier: tg,
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
