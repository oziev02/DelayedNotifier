package repository

import (
	"context"

	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
)

// CacheRepository интерфейс для кэширования
type CacheRepository interface {
	Set(ctx context.Context, notification *entity.Notification) error
	Get(ctx context.Context, id string) (*entity.Notification, error)
	Delete(ctx context.Context, id string) error
}
