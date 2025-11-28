package cache

import (
	"context"

	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
)

// NoOpCache пустая реализация кэша для случая, когда Redis недоступен
type NoOpCache struct{}

func (n *NoOpCache) Set(ctx context.Context, notification *entity.Notification) error {
	return nil
}

func (n *NoOpCache) Get(ctx context.Context, id string) (*entity.Notification, error) {
	return nil, nil
}

func (n *NoOpCache) Delete(ctx context.Context, id string) error {
	return nil
}
