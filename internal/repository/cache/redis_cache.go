package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
)

// RedisCache реализация кэша через Redis
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCache создает новый Redis кэш
func NewRedisCache(url string) (*RedisCache, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// Проверяем соединение
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
		ttl:    24 * time.Hour, // TTL по умолчанию
	}, nil
}

func (r *RedisCache) Set(ctx context.Context, notification *entity.Notification) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	key := "notification:" + notification.ID
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *RedisCache) Get(ctx context.Context, id string) (*entity.Notification, error) {
	key := "notification:" + id
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // не найдено в кэше
	}
	if err != nil {
		return nil, err
	}

	var notification entity.Notification
	if err := json.Unmarshal(data, &notification); err != nil {
		return nil, err
	}

	return &notification, nil
}

func (r *RedisCache) Delete(ctx context.Context, id string) error {
	key := "notification:" + id
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
