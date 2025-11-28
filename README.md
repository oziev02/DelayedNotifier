# DelayedNotifier

Сервис для отложенных уведомлений через очереди. Позволяет создавать уведомления, которые будут отправлены в указанное время через различные каналы (Email, Telegram).

## Возможности

- ✅ Создание отложенных уведомлений с указанием времени отправки
- ✅ Поддержка различных каналов отправки (Email, Telegram)
- ✅ Очередь уведомлений через RabbitMQ
- ✅ Автоматические повторные попытки с экспоненциальной задержкой при ошибках
- ✅ Кэширование статусов через Redis
- ✅ HTTP API для управления уведомлениями
- ✅ Веб-интерфейс для тестирования и мониторинга

## Требования

- Go 1.21+
- RabbitMQ
- Redis (опционально, но рекомендуется)

## Установка и запуск

### 1. Установка зависимостей

```bash
go mod download
```

### 2. Настройка переменных окружения

Создайте файл `.env` или установите переменные окружения:

```bash
# Порт HTTP сервера
export SERVER_PORT=8080

# RabbitMQ
export RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Redis (опционально)
export REDIS_URL=redis://localhost:6379

# Настройки Email (для реальной отправки)
export EMAIL_SMTP_HOST=smtp.gmail.com
export EMAIL_SMTP_PORT=587
export EMAIL_FROM=your-email@gmail.com

# Настройки Telegram (для реальной отправки)
export TELEGRAM_BOT_TOKEN=your-bot-token
export TELEGRAM_CHAT_ID=your-chat-id
```

### 3. Запуск RabbitMQ и Redis

#### Docker Compose (рекомендуется):

```bash
docker-compose up -d
```

Или вручную:

```bash
# RabbitMQ
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management

# Redis
docker run -d --name redis -p 6379:6379 redis:alpine
```

### 4. Запуск приложения

```bash
go run cmd/server/main.go
```

Сервис будет доступен по адресу: `http://localhost:8080`

## API

### POST /api/notify

Создание нового уведомления.

**Тело запроса:**
```json
{
  "user_id": "user123",
  "channel": "email",
  "recipient": "user@example.com",
  "subject": "Напоминание",
  "message": "Не забудьте о встрече!",
  "scheduled_at": "2024-12-31T12:00:00Z"
}
```

**Ответ:**
```json
{
  "id": "uuid",
  "user_id": "user123",
  "channel": "email",
  "recipient": "user@example.com",
  "subject": "Напоминание",
  "message": "Не забудьте о встрече!",
  "scheduled_at": "2024-12-31T12:00:00Z",
  "status": "scheduled",
  "created_at": "2024-12-30T10:00:00Z",
  "retry_count": 0
}
```

### GET /api/notify/:id

Получение статуса уведомления по ID.

**Ответ:**
```json
{
  "id": "uuid",
  "status": "sent",
  "scheduled_at": "2024-12-31T12:00:00Z",
  "sent_at": "2024-12-31T12:00:00Z",
  ...
}
```

### DELETE /api/notify/:id

Отмена запланированного уведомления.

**Ответ:**
```json
{
  "message": "уведомление отменено"
}
```

### GET /api/notify

Получение списка всех уведомлений.

**Ответ:**
```json
{
  "notifications": [
    {
      "id": "uuid",
      "status": "scheduled",
      ...
    }
  ]
}
```

## Веб-интерфейс

Доступен по адресу `http://localhost:8080/`. Позволяет:

- Создавать новые уведомления через форму
- Просматривать список всех уведомлений
- Отслеживать статусы уведомлений
- Отменять запланированные уведомления

## Архитектура

```
┌─────────────┐
│  HTTP API   │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐
│  Handlers   │────▶│  Storage    │
└──────┬──────┘     └─────────────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐
│   Queue     │────▶│   Worker    │
│ (RabbitMQ)  │     └──────┬──────┘
└─────────────┘            │
                           ▼
                    ┌─────────────┐
                    │   Senders   │
                    │ (Email/TG)  │
                    └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   Cache     │
                    │  (Redis)    │
                    └─────────────┘
```

## Статусы уведомлений

- `pending` - Ожидает обработки
- `scheduled` - Запланировано к отправке
- `sent` - Успешно отправлено
- `failed` - Ошибка отправки (превышено количество попыток)
- `cancelled` - Отменено пользователем

## Повторные попытки

При ошибке отправки уведомление автоматически повторяется с экспоненциальной задержкой:
- 1-я попытка: через 1 секунду
- 2-я попытка: через 2 секунды
- 3-я попытка: через 4 секунды
- 4-я попытка: через 8 секунд
- 5-я попытка: через 16 секунд

Максимальное количество попыток: 5 (настраивается в `cmd/server/main.go`)

## Разработка

### Структура проекта (Clean Architecture)

```
DelayedNotifier/
├── cmd/
│   └── server/
│       └── main.go              # Точка входа приложения
├── internal/
│   ├── domain/                   # Доменный слой
│   │   ├── entity/               # Сущности (Notification)
│   │   └── repository/           # Интерфейсы репозиториев
│   ├── usecase/                  # Слой бизнес-логики
│   │   ├── notification_usecase.go
│   │   └── worker_usecase.go
│   ├── delivery/                 # Слой доставки
│   │   └── http/                 # HTTP handlers
│   └── repository/               # Реализации репозиториев
│       ├── storage/              # Хранилище (in-memory)
│       ├── cache/                # Кэш (Redis, NoOp)
│       ├── queue/                # Очередь (RabbitMQ)
│       └── sender/               # Отправители (Email, Telegram)
├── pkg/
│   └── config/                   # Конфигурация
├── templates/                    # HTML шаблоны
├── static/                       # Статические файлы
├── docker-compose.yml
├── go.mod
└── README.md
```

### Архитектура

Проект следует принципам Clean Architecture:

- **Domain Layer** (`internal/domain/`) - содержит бизнес-сущности и интерфейсы репозиториев
- **Use Case Layer** (`internal/usecase/`) - содержит бизнес-логику приложения
- **Delivery Layer** (`internal/delivery/`) - содержит HTTP handlers и другие способы доставки
- **Repository Layer** (`internal/repository/`) - содержит реализации репозиториев (storage, cache, queue, sender)

Зависимости направлены внутрь: delivery → usecase → domain ← repository

## Лицензия

MIT

