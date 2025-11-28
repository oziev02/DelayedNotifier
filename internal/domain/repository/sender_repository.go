package repository

import (
	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
)

// SenderRepository интерфейс для отправки уведомлений
type SenderRepository interface {
	Send(notification *entity.Notification) error
	Supports(channel entity.ChannelType) bool
}
