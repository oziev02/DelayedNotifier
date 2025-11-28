package sender

import (
	"log"

	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
)

// TelegramSender отправляет уведомления через Telegram
type TelegramSender struct {
	botToken string
	chatID   string
}

// NewTelegramSender создает новый Telegram sender
func NewTelegramSender(botToken, chatID string) *TelegramSender {
	return &TelegramSender{
		botToken: botToken,
		chatID:   chatID,
	}
}

func (t *TelegramSender) Supports(channel entity.ChannelType) bool {
	return channel == entity.ChannelTelegram
}

func (t *TelegramSender) Send(notification *entity.Notification) error {
	// В реальном приложении здесь была бы отправка через Telegram Bot API
	// Для демонстрации просто логируем
	log.Printf("[TELEGRAM] Отправка уведомления ID=%s на %s: %s - %s",
		notification.ID, notification.Recipient, notification.Subject, notification.Message)

	// Симуляция отправки
	// В реальности здесь был бы код:
	// url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	// data := map[string]string{
	//     "chat_id": notification.Recipient,
	//     "text":    fmt.Sprintf("%s\n\n%s", notification.Subject, notification.Message),
	// }
	// resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	// ...

	return nil
}
