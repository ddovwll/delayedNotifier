package notifier_telegram

import (
	"context"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramNotifier struct {
	bot        *tgbotapi.BotAPI
	repository *TelegramNotifierRepository
}

func NewTelegramNotifier(bot *tgbotapi.BotAPI, repository *TelegramNotifierRepository) *TelegramNotifier {
	return &TelegramNotifier{
		bot:        bot,
		repository: repository,
	}
}

func (n *TelegramNotifier) Notify(ctx context.Context, recipient, message string) error {
	user, err := n.repository.GetByUsername(ctx, recipient)

	if err != nil || user == nil {
		return err
	}

	msg := tgbotapi.NewMessage(user.ChatId, message)
	_, err = n.bot.Send(msg)
	return err
}
