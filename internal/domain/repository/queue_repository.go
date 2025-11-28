package repository

import (
	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
)

// QueueRepository интерфейс для работы с очередью
type QueueRepository interface {
	Publish(notification *entity.Notification) error
	Consume() (<-chan *entity.Notification, error)
	Close() error
}
