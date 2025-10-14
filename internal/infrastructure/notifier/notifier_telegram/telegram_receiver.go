package notifier_telegram

import "github.com/google/uuid"

type TelegramReceiver struct {
	Id       uuid.UUID
	Username string
	ChatId   int64
}
