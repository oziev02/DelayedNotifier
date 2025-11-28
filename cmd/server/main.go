package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	httphandler "github.com/oziev02/DelayedNotifier/internal/delivery/http"
	"github.com/oziev02/DelayedNotifier/internal/domain/repository"
	cacherepo "github.com/oziev02/DelayedNotifier/internal/repository/cache"
	"github.com/oziev02/DelayedNotifier/internal/repository/queue"
	"github.com/oziev02/DelayedNotifier/internal/repository/sender"
	"github.com/oziev02/DelayedNotifier/internal/repository/storage"
	"github.com/oziev02/DelayedNotifier/internal/usecase"
	"github.com/oziev02/DelayedNotifier/pkg/config"
)

func main() {
	cfg := config.Load()

	// Инициализация хранилища
	notificationRepo := storage.NewInMemoryStorage()

	// Инициализация кэша (Redis)
	var cacheRepo repository.CacheRepository
	var err error
	redisCache, err := cacherepo.NewRedisCache(cfg.RedisURL)
	if err != nil {
		log.Printf("Предупреждение: не удалось подключиться к Redis (%v), работаем без кэша", err)
		cacheRepo = &cacherepo.NoOpCache{}
	} else {
		cacheRepo = redisCache
	}

	// Инициализация очереди (RabbitMQ)
	queueRepo, err := queue.NewRabbitMQQueue(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	}
	defer queueRepo.Close()

	// Инициализация отправителей
	emailSender := sender.NewEmailSender(cfg.EmailSMTPHost, cfg.EmailSMTPPort, cfg.EmailFrom)
	telegramSender := sender.NewTelegramSender(cfg.TelegramBotToken, cfg.TelegramChatID)
	senderManager := sender.NewSenderManager(emailSender, telegramSender)

	// Создаем usecase для уведомлений
	notificationUsecase := usecase.NewNotificationUsecase(notificationRepo, cacheRepo, queueRepo)

	// Создаем воркер
	workerUsecase := usecase.NewWorkerUsecase(queueRepo, notificationRepo, cacheRepo, senderManager, 5)

	// Запускаем воркера в отдельной горутине
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := workerUsecase.Start(ctx); err != nil {
			log.Printf("Ошибка воркера: %v", err)
		}
	}()

	// Настройка HTTP сервера
	handler := httphandler.NewHandler(notificationUsecase)

	router := gin.Default()

	// Статические файлы для UI
	router.Static("/static", "./static")
	router.LoadHTMLGlob("templates/*")

	// UI роуты
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	// API роуты
	api := router.Group("/api")
	{
		api.POST("/notify", handler.CreateNotification)
		api.GET("/notify/:id", handler.GetNotification)
		api.DELETE("/notify/:id", handler.DeleteNotification)
		api.GET("/notify", handler.ListNotifications)
	}

	// Запуск сервера
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		log.Printf("Сервер запущен на порту %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка сервера: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Завершение работы...")

	cancel() // Останавливаем воркера

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Ошибка при завершении сервера: %v", err)
	}

	log.Println("Сервер остановлен")
}
