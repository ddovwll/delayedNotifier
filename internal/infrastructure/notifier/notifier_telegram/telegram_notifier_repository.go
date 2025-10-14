package notifier_telegram

import (
	"context"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/retry"
)

type TelegramNotifierRepository struct {
	db            *dbpg.DB
	retryStrategy retry.Strategy
}

func NewTelegramNotifierRepository(db *dbpg.DB, retryStrategy retry.Strategy) *TelegramNotifierRepository {
	return &TelegramNotifierRepository{
		db:            db,
		retryStrategy: retryStrategy,
	}
}

func (n *TelegramNotifierRepository) Create(ctx context.Context, receiver *TelegramReceiver) error {
	receiver.Id = uuid.New()

	query := `
		INSERT INTO telegram_receivers (id, username, chat_id) VALUES ($1, $2, $3)
	`

	_, err := n.db.ExecWithRetry(ctx, n.retryStrategy, query,
		receiver.Id, receiver.Username, receiver.ChatId,
	)
	return err
}

func (n *TelegramNotifierRepository) GetByUsername(ctx context.Context, username string) (*TelegramReceiver, error) {
	query := `
	SELECT id, username, chat_id FROM telegram_receivers WHERE username = $1
	`

	row, err := n.db.QueryRowWithRetry(ctx, n.retryStrategy, query, username)
	if err != nil {
		return nil, err
	}

	var receiver TelegramReceiver
	err = row.Scan(
		&receiver.Id, &receiver.Username, &receiver.ChatId,
	)
	if err != nil {
		return nil, err
	}
	return &receiver, nil
}
