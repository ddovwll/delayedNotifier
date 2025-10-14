package notifier_telegram

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramBotListener struct {
	bot        *tgbotapi.BotAPI
	repository *TelegramNotifierRepository
}

func NewTelegramBotListener(bot *tgbotapi.BotAPI, repository *TelegramNotifierRepository) *TelegramBotListener {
	return &TelegramBotListener{
		bot:        bot,
		repository: repository,
	}
}

func (b *TelegramBotListener) SetupBot(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.bot.StopReceivingUpdates()
			for update := range updates {
				if update.Message == nil {
					continue
				}

				b.processUpdate(ctx, update)
			}
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			if update.Message == nil {
				continue
			}
			b.processUpdate(ctx, update)
		}
	}
}

func (b *TelegramBotListener) processUpdate(ctx context.Context, update tgbotapi.Update) {

	err := b.setUser(ctx, update.Message.Chat.UserName, update.Message.Chat.ID)
	if err != nil {
		log.Println(err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Error while add user for notifications")
		if _, err := b.bot.Send(msg); err != nil {
			log.Println("Error sending message:", err)
		}
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "User successfully added to notifications")
	if _, err := b.bot.Send(msg); err != nil {
		log.Println("Error sending message:", err)
	}
}

func (b *TelegramBotListener) setUser(ctx context.Context, username string, chatId int64) error {
	existing, _ := b.repository.GetByUsername(ctx, username)
	if existing != nil {
		return nil
	}

	user := TelegramReceiver{
		Username: username,
		ChatId:   chatId,
	}

	return b.repository.Create(ctx, &user)
}
