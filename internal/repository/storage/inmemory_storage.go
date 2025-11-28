package storage

import (
	"context"
	"sync"
	"time"

	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
	"github.com/oziev02/DelayedNotifier/internal/domain/repository"
)

// InMemoryStorage простая in-memory реализация для демонстрации
type InMemoryStorage struct {
	mu            sync.RWMutex
	notifications map[string]*entity.Notification
}

// NewInMemoryStorage создает новый in-memory storage
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		notifications: make(map[string]*entity.Notification),
	}
}

func (s *InMemoryStorage) Create(ctx context.Context, notification *entity.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications[notification.ID] = notification
	return nil
}

func (s *InMemoryStorage) GetByID(ctx context.Context, id string) (*entity.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	notification, exists := s.notifications[id]
	if !exists {
		return nil, repository.ErrNotFound
	}
	return notification, nil
}

func (s *InMemoryStorage) UpdateStatus(ctx context.Context, id string, status entity.NotificationStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	notification, exists := s.notifications[id]
	if !exists {
		return repository.ErrNotFound
	}
	notification.Status = status
	return nil
}

func (s *InMemoryStorage) UpdateSentAt(ctx context.Context, id string, sentAt interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	notification, exists := s.notifications[id]
	if !exists {
		return repository.ErrNotFound
	}
	if t, ok := sentAt.(time.Time); ok {
		notification.SentAt = &t
	}
	return nil
}

func (s *InMemoryStorage) IncrementRetryCount(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	notification, exists := s.notifications[id]
	if !exists {
		return repository.ErrNotFound
	}
	notification.RetryCount++
	return nil
}

func (s *InMemoryStorage) GetAll(ctx context.Context) ([]*entity.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	notifications := make([]*entity.Notification, 0, len(s.notifications))
	for _, notification := range s.notifications {
		notifications = append(notifications, notification)
	}
	return notifications, nil
}

func (s *InMemoryStorage) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.notifications[id]
	if !exists {
		return repository.ErrNotFound
	}
	delete(s.notifications, id)
	return nil
}
