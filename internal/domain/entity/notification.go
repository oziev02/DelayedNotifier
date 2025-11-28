package entity

import (
	"time"
)

// NotificationStatus представляет статус уведомления
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusScheduled NotificationStatus = "scheduled"
	StatusSent      NotificationStatus = "sent"
	StatusFailed    NotificationStatus = "failed"
	StatusCancelled NotificationStatus = "cancelled"
)

// ChannelType представляет тип канала отправки
type ChannelType string

const (
	ChannelEmail    ChannelType = "email"
	ChannelTelegram ChannelType = "telegram"
)

// Notification представляет уведомление
type Notification struct {
	ID          string             `json:"id"`
	UserID      string             `json:"user_id"`
	Channel     ChannelType        `json:"channel"`
	Recipient   string             `json:"recipient"` // email или telegram ID
	Subject     string             `json:"subject,omitempty"`
	Message     string             `json:"message"`
	ScheduledAt time.Time          `json:"scheduled_at"`
	Status      NotificationStatus `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	SentAt      *time.Time         `json:"sent_at,omitempty"`
	RetryCount  int                `json:"retry_count"`
}

// NotificationRequest представляет запрос на создание уведомления
type NotificationRequest struct {
	UserID      string    `json:"user_id" binding:"required"`
	Channel     string    `json:"channel" binding:"required,oneof=email telegram"`
	Recipient   string    `json:"recipient" binding:"required"`
	Subject     string    `json:"subject,omitempty"`
	Message     string    `json:"message" binding:"required"`
	ScheduledAt time.Time `json:"scheduled_at" binding:"required"`
}
