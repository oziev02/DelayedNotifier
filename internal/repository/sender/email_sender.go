package sender

import (
	"log"

	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
)

// EmailSender отправляет уведомления по email
type EmailSender struct {
	smtpHost string
	smtpPort string
	from     string
}

// NewEmailSender создает новый email sender
func NewEmailSender(smtpHost, smtpPort, from string) *EmailSender {
	return &EmailSender{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		from:     from,
	}
}

func (e *EmailSender) Supports(channel entity.ChannelType) bool {
	return channel == entity.ChannelEmail
}

func (e *EmailSender) Send(notification *entity.Notification) error {
	// В реальном приложении здесь была бы отправка через SMTP
	// Для демонстрации просто логируем
	log.Printf("[EMAIL] Отправка уведомления ID=%s на %s: %s - %s",
		notification.ID, notification.Recipient, notification.Subject, notification.Message)

	// Симуляция отправки
	// В реальности здесь был бы код:
	// auth := smtp.PlainAuth("", username, password, e.smtpHost)
	// msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s",
	//     notification.Recipient, notification.Subject, notification.Message))
	// return smtp.SendMail(fmt.Sprintf("%s:%s", e.smtpHost, e.smtpPort), auth, e.from,
	//     []string{notification.Recipient}, msg)

	return nil
}
