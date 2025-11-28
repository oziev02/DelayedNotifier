package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/oziev02/DelayedNotifier/internal/domain/entity"
	"github.com/oziev02/DelayedNotifier/internal/domain/repository"
	"github.com/oziev02/DelayedNotifier/internal/usecase"
)

// Handler содержит зависимости для обработчиков
type Handler struct {
	notificationUsecase *usecase.NotificationUsecase
}

// NewHandler создает новый обработчик
func NewHandler(notificationUsecase *usecase.NotificationUsecase) *Handler {
	return &Handler{
		notificationUsecase: notificationUsecase,
	}
}

// CreateNotification создает новое уведомление
func (h *Handler) CreateNotification(c *gin.Context) {
	var req entity.NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification, err := h.notificationUsecase.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

// GetNotification получает статус уведомления
func (h *Handler) GetNotification(c *gin.Context) {
	id := c.Param("id")

	notification, err := h.notificationUsecase.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "уведомление не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка получения уведомления"})
		return
	}

	c.JSON(http.StatusOK, notification)
}

// DeleteNotification отменяет уведомление
func (h *Handler) DeleteNotification(c *gin.Context) {
	id := c.Param("id")

	if err := h.notificationUsecase.Cancel(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "уведомление не найдено"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "уведомление отменено"})
}

// ListNotifications получает список всех уведомлений (для UI)
func (h *Handler) ListNotifications(c *gin.Context) {
	notifications, err := h.notificationUsecase.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка получения списка уведомлений"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}
