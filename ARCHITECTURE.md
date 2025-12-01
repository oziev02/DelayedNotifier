# Архитектура приложения DelayedNotifier

## Общий обзор

DelayedNotifier - это приложение для отправки отложенных уведомлений через различные каналы (Email, Telegram). Приложение использует архитектуру Clean Architecture с разделением на слои: Delivery, Usecase, Repository.

---

## Полный путь выполнения запроса

### 1️⃣ ТОЧКА ВХОДА: `cmd/server/main.go`

**Что происходит при запуске:**

```25:145:cmd/server/main.go
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

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := workerUsecase.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
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
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка сервера: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Завершение работы...")

	// Останавливаем воркера
	cancel()

	// Ожидаем завершения воркера с таймаутом
	workerDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(workerDone)
	}()

	// Graceful shutdown сервера
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Printf("Ошибка при завершении сервера: %v", err)
	} else {
		log.Println("Сервер остановлен")
	}

	// Ожидаем завершения воркера или таймаут
	select {
	case <-workerDone:
		log.Println("Воркер остановлен")
	case <-time.After(5 * time.Second):
		log.Println("Таймаут ожидания воркера")
	}

	log.Println("Приложение завершено")
}
```

**Инициализируются:**
- ✅ Конфигурация из переменных окружения (`pkg/config/config.go`)
- ✅ Хранилище уведомлений (`InMemoryStorage`)
- ✅ Кэш (Redis или NoOp, если Redis недоступен)
- ✅ Очередь (RabbitMQ)
- ✅ Отправители (Email и Telegram)
- ✅ Менеджер отправителей (`SenderManager`)
- ✅ Usecase для уведомлений
- ✅ Воркер (запускается в отдельной горутине)
- ✅ HTTP сервер (Gin)

---

### 2️⃣ СОЗДАНИЕ УВЕДОМЛЕНИЯ: HTTP запрос `POST /api/notify`

#### Шаг 1: HTTP Handler получает запрос

**Файл:** `internal/delivery/http/handler.go`

```26:40:internal/delivery/http/handler.go
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
```

**Что происходит:**
1. Gin роутер (`cmd/server/main.go:90`) перенаправляет запрос на `handler.CreateNotification`
2. Handler парсит JSON тело запроса в структуру `NotificationRequest`
3. Handler вызывает `notificationUsecase.Create()`

---

#### Шаг 2: Usecase обрабатывает бизнес-логику

**Файл:** `internal/usecase/notification_usecase.go`

```34:68:internal/usecase/notification_usecase.go
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

	// Кэшируем (ошибка кэширования не критична)
	_ = u.cacheRepo.Set(ctx, notification)

	// Публикуем в очередь
	if err := u.queueRepo.Publish(notification); err != nil {
		return nil, err
	}

	return notification, nil
}
```

**Что происходит:**
1. ✅ **Валидация:** Проверка, что время отправки в будущем
2. ✅ **Создание сущности:** Формирование объекта `Notification` с уникальным ID
3. ✅ **Сохранение в хранилище:** Запись в `InMemoryStorage` (`internal/repository/storage/inmemory_storage.go`)
4. ✅ **Кэширование:** Сохранение в Redis (если доступен)
5. ✅ **Публикация в очередь:** Отправка в RabbitMQ (`internal/repository/queue/rabbitmq_queue.go`)

---

#### Шаг 3: Сохранение в хранилище

**Файл:** `internal/repository/storage/inmemory_storage.go`

```25:30:internal/repository/storage/inmemory_storage.go
func (s *InMemoryStorage) Create(ctx context.Context, notification *entity.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications[notification.ID] = notification
	return nil
}
```

**Что происходит:**
- Уведомление сохраняется в map в памяти (thread-safe благодаря `sync.RWMutex`)

---

#### Шаг 4: Публикация в очередь RabbitMQ

**Файл:** `internal/repository/queue/rabbitmq_queue.go`

```56:79:internal/repository/queue/rabbitmq_queue.go
// Publish публикует уведомление в очередь
func (r *RabbitMQQueue) Publish(notification *entity.Notification) error {
	body, err := json.Marshal(notification)
	if err != nil {
		return err
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
```

**Что происходит:**
- Уведомление сериализуется в JSON
- Публикуется в очередь RabbitMQ с именем "notifications"
- В заголовках сохраняется время планируемой отправки

---

### 3️⃣ ОБРАБОТКА УВЕДОМЛЕНИЯ: Worker

#### Шаг 5: Worker читает из очереди

**Файл:** `internal/usecase/worker_usecase.go`

**Запуск воркера происходит при старте приложения:**

```40:60:internal/usecase/worker_usecase.go
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
```

**Что происходит:**
1. Worker запускается в отдельной горутине при старте приложения (`cmd/server/main.go:66-71`)
2. Worker подписывается на канал сообщений из RabbitMQ
3. Каждое уведомление обрабатывается в отдельной горутине (`go w.processNotification()`)

---

#### Шаг 6: RabbitMQ Consume получает сообщения

**Файл:** `internal/repository/queue/rabbitmq_queue.go`

```81:122:internal/repository/queue/rabbitmq_queue.go
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
				_ = msg.Nack(false, false) // не переотправлять
				continue
			}

			// Проверяем, наступило ли время отправки
			if time.Now().Before(notification.ScheduledAt) {
				// Еще рано, возвращаем в очередь с небольшой задержкой
				_ = msg.Nack(false, true) // переотправить
				time.Sleep(time.Second)
				continue
			}

			notifications <- &notification
			_ = msg.Ack(false)
		}
		close(notifications)
	}()

	return notifications, nil
}
```

**Что происходит:**
1. Создается канал для получения сообщений из RabbitMQ
2. В отдельной горутине десериализуются JSON сообщения
3. **Проверка времени:** Если время отправки еще не наступило, сообщение возвращается в очередь
4. Если время пришло, уведомление отправляется в канал `notifications`

---

#### Шаг 7: Обработка уведомления в Worker

**Файл:** `internal/usecase/worker_usecase.go`

```62:107:internal/usecase/worker_usecase.go
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
```

**Что происходит:**
1. ✅ **Проверка отмены:** Проверяется, не было ли уведомление отменено пользователем
2. ✅ **Повторная проверка времени:** Дополнительная проверка времени отправки
3. ✅ **Отправка:** Вызов `sender.Send()` через SenderManager
4. ✅ **Обработка ошибок:** При ошибке запускается механизм retry
5. ✅ **Обновление статуса:** При успехе статус меняется на `StatusSent`
6. ✅ **Обновление кэша:** Кэш обновляется новой информацией

---

#### Шаг 8: Отправка через SenderManager

**Файл:** `internal/repository/sender/sender_manager.go`

```20:28:internal/repository/sender/sender_manager.go
// Send отправляет уведомление через подходящий канал
func (sm *SenderManager) Send(notification *entity.Notification) error {
	for _, sender := range sm.senders {
		if sender.Supports(notification.Channel) {
			return sender.Send(notification)
		}
	}
	return fmt.Errorf("нет доступного отправителя для канала %s", notification.Channel)
}
```

**Что происходит:**
1. SenderManager перебирает всех отправителей (Email, Telegram)
2. Находит подходящего через метод `Supports()`
3. Вызывает метод `Send()` выбранного отправителя

---

#### Шаг 9: Конкретная отправка (Email или Telegram)

**Email Sender:**
```29:44:internal/repository/sender/email_sender.go
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
```

**Telegram Sender:**
```27:44:internal/repository/sender/telegram_sender.go
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
```

**Что происходит:**
- В текущей версии отправка только логируется (симуляция)
- В реальном приложении здесь была бы реальная отправка через SMTP или Telegram Bot API

---

### 4️⃣ МЕХАНИЗМ ПОВТОРНОЙ ОТПРАВКИ (Retry)

**Файл:** `internal/usecase/worker_usecase.go`

```109:137:internal/usecase/worker_usecase.go
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
```

**Что происходит:**
1. Проверяется количество попыток (максимум 5)
2. Если превышен лимит, статус меняется на `StatusFailed`
3. Вычисляется экспоненциальная задержка: 2^retryCount секунд
4. Уведомление публикуется обратно в очередь с новым временем отправки

---

## Другие операции

### Получение уведомления: `GET /api/notify/:id`

**Путь выполнения:**
1. `handler.GetNotification()` → 
2. `usecase.GetByID()` → 
3. Проверка кэша → 
4. Если нет в кэше, запрос в хранилище → 
5. Обновление кэша → 
6. Возврат результата

**Файлы:**
- `internal/delivery/http/handler.go:42-57`
- `internal/usecase/notification_usecase.go:70-88`

---

### Отмена уведомления: `DELETE /api/notify/:id`

**Путь выполнения:**
1. `handler.DeleteNotification()` → 
2. `usecase.Cancel()` → 
3. Проверка статуса (нельзя отменить отправленное) → 
4. Обновление статуса на `StatusCancelled` → 
5. Удаление из кэша → 
6. Удаление из хранилища

**Файлы:**
- `internal/delivery/http/handler.go:59-73`
- `internal/usecase/notification_usecase.go:95-122`

---

## Архитектурные слои

### 📦 Domain Layer (Доменный слой)
- `internal/domain/entity/` - Сущности (Notification)
- `internal/domain/repository/` - Интерфейсы репозиториев

### 🔧 Usecase Layer (Слой бизнес-логики)
- `internal/usecase/notification_usecase.go` - Логика работы с уведомлениями
- `internal/usecase/worker_usecase.go` - Логика обработки воркера

### 🚀 Delivery Layer (Слой доставки)
- `internal/delivery/http/handler.go` - HTTP обработчики

### 💾 Repository Layer (Слой данных)
- `internal/repository/storage/` - Хранилище (InMemory)
- `internal/repository/cache/` - Кэш (Redis/NoOp)
- `internal/repository/queue/` - Очередь (RabbitMQ)
- `internal/repository/sender/` - Отправители (Email/Telegram)

---

## Диаграмма потока данных

```
Пользователь
    ↓
HTTP запрос (POST /api/notify)
    ↓
Gin Router
    ↓
HTTP Handler (handler.CreateNotification)
    ↓
NotificationUsecase.Create()
    ├──→ Валидация
    ├──→ Создание сущности
    ├──→ Сохранение в InMemoryStorage
    ├──→ Кэширование в Redis
    └──→ Публикация в RabbitMQ
         ↓
    RabbitMQ Queue ("notifications")
         ↓
WorkerUsecase (читает из очереди)
    ↓
processNotification()
    ├──→ Проверка отмены
    ├──→ Проверка времени
    └──→ SenderManager.Send()
         ├──→ EmailSender (если channel=email)
         └──→ TelegramSender (если channel=telegram)
              ↓
    Успех → Обновление статуса на "sent"
         → Обновление кэша
    Ошибка → Retry механизм (экспоненциальная задержка)
         → Повторная публикация в очередь
```

---

## Зависимости между компонентами

```
main.go
├── config.Load() → Конфигурация
├── InMemoryStorage → NotificationRepository
├── RedisCache/NoOpCache → CacheRepository
├── RabbitMQQueue → QueueRepository
├── EmailSender + TelegramSender → SenderManager
├── NotificationUsecase
│   ├── NotificationRepository
│   ├── CacheRepository
│   └── QueueRepository
├── WorkerUsecase
│   ├── QueueRepository
│   ├── NotificationRepository
│   ├── CacheRepository
│   └── SenderManager
└── HTTP Handler
    └── NotificationUsecase
```

---

## Ключевые особенности архитектуры

1. **Clean Architecture:** Разделение на слои с зависимостями, направленными внутрь
2. **Dependency Injection:** Все зависимости передаются через конструкторы
3. **Интерфейсы:** Использование интерфейсов для абстракции (Repository pattern)
4. **Асинхронная обработка:** Worker работает в отдельной горутине
5. **Очередь сообщений:** Использование RabbitMQ для надежной доставки
6. **Кэширование:** Redis для быстрого доступа к данным
7. **Retry механизм:** Экспоненциальная задержка при ошибках
8. **Graceful shutdown:** Корректное завершение работы с ожиданием завершения воркера

---

## Важные файлы и их роль

| Файл | Роль |
|------|------|
| `cmd/server/main.go` | Точка входа, инициализация всех компонентов |
| `internal/delivery/http/handler.go` | HTTP обработчики (принимают запросы) |
| `internal/usecase/notification_usecase.go` | Бизнес-логика работы с уведомлениями |
| `internal/usecase/worker_usecase.go` | Логика обработки уведомлений из очереди |
| `internal/domain/entity/notification.go` | Доменная модель (Notification) |
| `internal/repository/storage/inmemory_storage.go` | Хранилище в памяти |
| `internal/repository/queue/rabbitmq_queue.go` | Интеграция с RabbitMQ |
| `internal/repository/sender/sender_manager.go` | Управление отправителями |

