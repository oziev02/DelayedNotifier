package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
	"github.com/oziev02/DelayedNotifier/internal/domain/repository"
)

// NotificationUsecase содержит бизнес-логику для работы с уведомлениями
type NotificationUsecase struct {
	notificationRepo repository.NotificationRepository
	cacheRepo        repository.CacheRepository
	queueRepo        repository.QueueRepository
}

// NewNotificationUsecase создает новый usecase
func NewNotificationUsecase(
	notificationRepo repository.NotificationRepository,
	cacheRepo repository.CacheRepository,
	queueRepo repository.QueueRepository,
) *NotificationUsecase {
	return &NotificationUsecase{
		notificationRepo: notificationRepo,
		cacheRepo:        cacheRepo,
		queueRepo:        queueRepo,
	}
}

// Create создает новое уведомление
func (u *NotificationUsecase) Create(ctx context.Context, req *entity.NotificationRequest) (*entity.Notification, error) {
	// Проверяем, что время отправки в будущем
	if req.ScheduledAt.Before(time.Now()) {
		return nil, errors.New("время отправки должно быть в будущем")
	}

	// Создаем уведомление
	notification := &entity.Notification{
		ID:          uuid.New().String(),
		UserID:      req.UserID,
		Channel:     entity.ChannelType(req.Channel),
		Recipient:   req.Recipient,
		Subject:     req.Subject,
		Message:     req.Message,
		ScheduledAt: req.ScheduledAt,
		Status:      entity.StatusScheduled,
		CreatedAt:   time.Now(),
		RetryCount:  0,
	}

	// Сохраняем в хранилище
	if err := u.notificationRepo.Create(ctx, notification); err != nil {
		return nil, err
	}

	// Кэшируем
	if err := u.cacheRepo.Set(ctx, notification); err != nil {
		// Логируем, но не прерываем выполнение
	}

	// Публикуем в очередь
	if err := u.queueRepo.Publish(notification); err != nil {
		return nil, err
	}

	return notification, nil
}

// GetByID получает уведомление по ID
func (u *NotificationUsecase) GetByID(ctx context.Context, id string) (*entity.Notification, error) {
	// Сначала проверяем кэш
	notification, err := u.cacheRepo.Get(ctx, id)
	if err == nil && notification != nil {
		return notification, nil
	}

	// Если нет в кэше, берем из хранилища
	notification, err = u.notificationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Обновляем кэш
	if err := u.cacheRepo.Set(ctx, notification); err != nil {
		// Логируем, но не прерываем выполнение
	}

	return notification, nil
}

// GetAll получает список всех уведомлений
func (u *NotificationUsecase) GetAll(ctx context.Context) ([]*entity.Notification, error) {
	return u.notificationRepo.GetAll(ctx)
}

// Cancel отменяет уведомление
func (u *NotificationUsecase) Cancel(ctx context.Context, id string) error {
	// Проверяем существование
	notification, err := u.notificationRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Проверяем, можно ли отменить
	if notification.Status == entity.StatusSent {
		return errors.New("нельзя отменить уже отправленное уведомление")
	}

	// Обновляем статус
	if err := u.notificationRepo.UpdateStatus(ctx, id, entity.StatusCancelled); err != nil {
		return err
	}

	// Удаляем из кэша
	if err := u.cacheRepo.Delete(ctx, id); err != nil {
		// Логируем, но не прерываем выполнение
	}

	// Удаляем из хранилища
	if err := u.notificationRepo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}
