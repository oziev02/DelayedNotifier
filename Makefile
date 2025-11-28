.PHONY: help build run test clean docker-up docker-down lint fmt vet deps install

# Переменные
BINARY_NAME=delayednotifier
MAIN_PATH=./cmd/server
DOCKER_COMPOSE=docker-compose

# Цвета для вывода
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

help: ## Показать справку по командам
	@echo "$(GREEN)Доступные команды:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

deps: ## Загрузить зависимости
	@echo "$(GREEN)Загрузка зависимостей...$(NC)"
	go mod download
	go mod tidy

install: deps ## Установить зависимости и проверить
	@echo "$(GREEN)Установка зависимостей...$(NC)"
	go mod verify

build: ## Собрать приложение
	@echo "$(GREEN)Сборка приложения...$(NC)"
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Готово: bin/$(BINARY_NAME)$(NC)"

run: ## Запустить приложение
	@echo "$(GREEN)Запуск приложения...$(NC)"
	go run $(MAIN_PATH)

test: ## Запустить тесты
	@echo "$(GREEN)Запуск тестов...$(NC)"
	go test -v ./...

test-coverage: ## Запустить тесты с покрытием
	@echo "$(GREEN)Запуск тестов с покрытием...$(NC)"
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Отчет сохранен в coverage.html$(NC)"

lint: ## Запустить линтер
	@echo "$(GREEN)Проверка кода линтером...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)golangci-lint не установлен. Установите: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi

fmt: ## Форматировать код
	@echo "$(GREEN)Форматирование кода...$(NC)"
	go fmt ./...
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
		echo "$(GREEN)Импорты отсортированы$(NC)"; \
	else \
		echo "$(YELLOW)goimports не установлен. Установите: go install golang.org/x/tools/cmd/goimports@latest$(NC)"; \
	fi

vet: ## Запустить go vet
	@echo "$(GREEN)Проверка кода go vet...$(NC)"
	go vet ./...

clean: ## Очистить скомпилированные файлы
	@echo "$(GREEN)Очистка...$(NC)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean

docker-up: ## Запустить Docker Compose (RabbitMQ и Redis)
	@echo "$(GREEN)Запуск Docker Compose...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)RabbitMQ: http://localhost:15672 (guest/guest)$(NC)"
	@echo "$(GREEN)Redis: localhost:6379$(NC)"

docker-down: ## Остановить Docker Compose
	@echo "$(GREEN)Остановка Docker Compose...$(NC)"
	$(DOCKER_COMPOSE) down

docker-logs: ## Показать логи Docker Compose
	$(DOCKER_COMPOSE) logs -f

docker-restart: docker-down docker-up ## Перезапустить Docker Compose

dev: docker-up run ## Запустить окружение разработки (Docker + приложение)

check: fmt vet lint ## Проверить код (форматирование, vet, линтер)

all: clean deps build test ## Полная сборка: очистка, зависимости, сборка, тесты

