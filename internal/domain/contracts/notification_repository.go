package contracts

import (
	"context"
	"delayedNotifier/internal/domain/models"

	"github.com/google/uuid"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *models.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	Update(ctx context.Context, n *models.Notification) error
	Delete(ctx context.Context, id uuid.UUID) error
}
