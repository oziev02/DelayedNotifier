package sender

import (
	"fmt"

	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
	"github.com/oziev02/DelayedNotifier/internal/domain/repository"
)

// SenderManager управляет различными отправителями
type SenderManager struct {
	senders []repository.SenderRepository
}

// NewSenderManager создает новый менеджер отправителей
func NewSenderManager(senders ...repository.SenderRepository) *SenderManager {
	return &SenderManager{senders: senders}
}

// Send отправляет уведомление через подходящий канал
func (sm *SenderManager) Send(notification *entity.Notification) error {
	for _, sender := range sm.senders {
		if sender.Supports(notification.Channel) {
			return sender.Send(notification)
		}
	}
	return fmt.Errorf("нет доступного отправителя для канала %s", notification.Channel)
}
