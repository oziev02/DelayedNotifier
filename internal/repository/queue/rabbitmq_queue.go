package queue

import (
	"encoding/json"
	"log"
	"time"

	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQQueue реализация очереди через RabbitMQ
type RabbitMQQueue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

// NewRabbitMQQueue создает новое подключение к RabbitMQ
func NewRabbitMQQueue(url string) (*RabbitMQQueue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Объявляем очередь с поддержкой TTL и dead letter exchange
	q, err := ch.QueueDeclare(
		"notifications", // имя очереди
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		amqp.Table{
			"x-message-ttl": int32(86400000), // 24 часа в миллисекундах
		},
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitMQQueue{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

// Publish публикует уведомление в очередь
func (r *RabbitMQQueue) Publish(notification *entity.Notification) error {
	body, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// Вычисляем задержку до времени отправки
	delay := time.Until(notification.ScheduledAt)
	if delay < 0 {
		delay = 0
	}

	// Используем RabbitMQ delayed message plugin или альтернативный подход
	// Для простоты используем обычную публикацию, а задержку обработаем в consumer
	return r.channel.Publish(
		"",           // exchange
		r.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Headers: amqp.Table{
				"x-scheduled-at": notification.ScheduledAt.Unix(),
			},
		},
	)
}

// Consume начинает потребление сообщений из очереди
func (r *RabbitMQQueue) Consume() (<-chan *entity.Notification, error) {
	msgs, err := r.channel.Consume(
		r.queue.Name, // queue
		"",           // consumer
		false,        // auto-ack (false, чтобы вручную подтверждать)
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return nil, err
	}

	notifications := make(chan *entity.Notification, 100)

	go func() {
		for msg := range msgs {
			var notification entity.Notification
			if err := json.Unmarshal(msg.Body, &notification); err != nil {
				log.Printf("Ошибка декодирования уведомления: %v", err)
				msg.Nack(false, false) // не переотправлять
				continue
			}

			// Проверяем, наступило ли время отправки
			if time.Now().Before(notification.ScheduledAt) {
				// Еще рано, возвращаем в очередь с небольшой задержкой
				msg.Nack(false, true) // переотправить
				time.Sleep(time.Second)
				continue
			}

			notifications <- &notification
			msg.Ack(false)
		}
		close(notifications)
	}()

	return notifications, nil
}

// Close закрывает соединение
func (r *RabbitMQQueue) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}
