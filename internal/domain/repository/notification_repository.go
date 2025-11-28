package repository

import (
	"context"
	"errors"

	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
)

var (
	ErrNotFound = errors.New("notification not found")
)

// NotificationRepository интерфейс для хранения уведомлений
type NotificationRepository interface {
	Create(ctx context.Context, notification *entity.Notification) error
	GetByID(ctx context.Context, id string) (*entity.Notification, error)
	GetAll(ctx context.Context) ([]*entity.Notification, error)
	UpdateStatus(ctx context.Context, id string, status entity.NotificationStatus) error
	UpdateSentAt(ctx context.Context, id string, sentAt interface{}) error
	IncrementRetryCount(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}
