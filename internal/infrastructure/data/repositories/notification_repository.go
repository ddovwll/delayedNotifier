package repositories

import (
	"context"
	"time"

	"delayedNotifier/internal/domain/models"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/retry"
)

type NotificationRepository struct {
	db            *dbpg.DB
	retryStrategy retry.Strategy
}

func NewNotificationRepository(db *dbpg.DB, strategy retry.Strategy) *NotificationRepository {
	return &NotificationRepository{
		db:            db,
		retryStrategy: strategy,
	}
}

func (r *NotificationRepository) Create(ctx context.Context, n *models.Notification) error {
	n.ID = uuid.New()
	n.CreatedAt = time.Now().UTC()
	n.UpdatedAt = n.CreatedAt

	query := `
		INSERT INTO notifications (
			id, channel, recipient, message, scheduled_at, status, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`

	_, err := r.db.ExecWithRetry(ctx, r.retryStrategy, query,
		n.ID, n.Channel, n.Recipient, n.Message,
		n.ScheduledAt, n.Status, n.CreatedAt, n.UpdatedAt,
	)
	return err
}

func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	query := `
		SELECT id, channel, recipient, message, scheduled_at, status, created_at, updated_at
		FROM notifications
		WHERE id = $1
	`

	row, err := r.db.QueryRowWithRetry(ctx, r.retryStrategy, query, id)
	if err != nil {
		return nil, err
	}

	var n models.Notification
	err = row.Scan(
		&n.ID, &n.Channel, &n.Recipient, &n.Message,
		&n.ScheduledAt, &n.Status, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NotificationRepository) Update(ctx context.Context, n *models.Notification) error {
	n.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE notifications
		SET channel=$2, recipient=$3, message=$4,
			scheduled_at=$5, status=$6, updated_at=$7
		WHERE id=$1
	`

	_, err := r.db.ExecWithRetry(ctx, r.retryStrategy, query,
		n.ID, n.Channel, n.Recipient, n.Message,
		n.ScheduledAt, n.Status, n.UpdatedAt,
	)
	return err
}

func (r *NotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id=$1`
	_, err := r.db.ExecWithRetry(ctx, r.retryStrategy, query, id)
	return err
}
