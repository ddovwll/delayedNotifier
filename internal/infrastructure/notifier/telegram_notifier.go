package notifier

import (
	"strconv"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramNotifier struct {
	bot *tgbotapi.BotAPI
}

func (n *TelegramNotifier) Notify(recipient, message string) error {
	chatID, err := strconv.ParseInt(recipient, 10, 64)
	if err != nil {
		return err
	}

	msg := tgbotapi.NewMessage(chatID, message)
	_, err = n.bot.Send(msg)
	return err
}
