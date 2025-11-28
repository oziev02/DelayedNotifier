package usecase

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
	"github.com/oziev02/DelayedNotifier/internal/domain/repository"
	"github.com/oziev02/DelayedNotifier/internal/repository/sender"
)

// WorkerUsecase обрабатывает уведомления из очереди
type WorkerUsecase struct {
	queueRepo        repository.QueueRepository
	notificationRepo repository.NotificationRepository
	cacheRepo        repository.CacheRepository
	sender           *sender.SenderManager
	maxRetries       int
}

// NewWorkerUsecase создает нового воркера
func NewWorkerUsecase(
	q repository.QueueRepository,
	s repository.NotificationRepository,
	c repository.CacheRepository,
	sender *sender.SenderManager,
	maxRetries int,
) *WorkerUsecase {
	return &WorkerUsecase{
		queueRepo:        q,
		notificationRepo: s,
		cacheRepo:        c,
		sender:           sender,
		maxRetries:       maxRetries,
	}
}

// Start запускает воркера
func (w *WorkerUsecase) Start(ctx context.Context) error {
	notifications, err := w.queueRepo.Consume()
	if err != nil {
		return err
	}

	log.Println("Воркер запущен, ожидание уведомлений...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case notification, ok := <-notifications:
			if !ok {
				return nil
			}
			go w.processNotification(ctx, notification)
		}
	}
}

// processNotification обрабатывает одно уведомление
func (w *WorkerUsecase) processNotification(ctx context.Context, notification *entity.Notification) {
	// Проверяем, не отменено ли уведомление
	stored, err := w.notificationRepo.GetByID(ctx, notification.ID)
	if err != nil {
		log.Printf("Ошибка получения уведомления %s: %v", notification.ID, err)
		return
	}

	if stored.Status == entity.StatusCancelled {
		log.Printf("Уведомление %s отменено, пропускаем", notification.ID)
		return
	}

	// Проверяем, наступило ли время отправки
	if time.Now().Before(notification.ScheduledAt) {
		log.Printf("Время отправки еще не наступило для %s, ждем...", notification.ID)
		return
	}

	// Пытаемся отправить
	err = w.sender.Send(notification)
	if err != nil {
		log.Printf("Ошибка отправки уведомления %s: %v", notification.ID, err)
		w.handleRetry(ctx, notification, err)
		return
	}

	// Успешно отправлено
	now := time.Now()
	if err := w.notificationRepo.UpdateStatus(ctx, notification.ID, entity.StatusSent); err != nil {
		log.Printf("Ошибка обновления статуса: %v", err)
	}
	if err := w.notificationRepo.UpdateSentAt(ctx, notification.ID, now); err != nil {
		log.Printf("Ошибка обновления времени отправки: %v", err)
	}

	// Обновляем кэш
	notification.Status = entity.StatusSent
	notification.SentAt = &now
	if err := w.cacheRepo.Set(ctx, notification); err != nil {
		log.Printf("Ошибка обновления кэша: %v", err)
	}

	log.Printf("Уведомление %s успешно отправлено", notification.ID)
}

// handleRetry обрабатывает повторную попытку с экспоненциальной задержкой
func (w *WorkerUsecase) handleRetry(ctx context.Context, notification *entity.Notification, _ error) {
	if notification.RetryCount >= w.maxRetries {
		log.Printf("Достигнуто максимальное количество попыток для %s", notification.ID)
		if err := w.notificationRepo.UpdateStatus(ctx, notification.ID, entity.StatusFailed); err != nil {
			log.Printf("Ошибка обновления статуса: %v", err)
		}
		return
	}

	// Увеличиваем счетчик попыток
	if err := w.notificationRepo.IncrementRetryCount(ctx, notification.ID); err != nil {
		log.Printf("Ошибка увеличения счетчика попыток: %v", err)
	}

	// Вычисляем экспоненциальную задержку: 2^retryCount секунд
	delay := time.Duration(math.Pow(2, float64(notification.RetryCount))) * time.Second
	log.Printf("Повторная попытка отправки %s через %v (попытка %d/%d)",
		notification.ID, delay, notification.RetryCount+1, w.maxRetries)

	// Обновляем время отправки для повторной попытки
	notification.ScheduledAt = time.Now().Add(delay)
	notification.RetryCount++

	// Публикуем обратно в очередь
	if err := w.queueRepo.Publish(notification); err != nil {
		log.Printf("Ошибка повторной публикации уведомления %s: %v", notification.ID, err)
	}
}
