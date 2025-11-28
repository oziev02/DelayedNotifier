package config

import (
	"os"
)

// Config содержит конфигурацию приложения
type Config struct {
	ServerPort       string
	RabbitMQURL      string
	RedisURL         string
	EmailSMTPHost    string
	EmailSMTPPort    string
	EmailFrom        string
	TelegramBotToken string
	TelegramChatID   string
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	return &Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		RabbitMQURL:      getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379"),
		EmailSMTPHost:    getEnv("EMAIL_SMTP_HOST", "smtp.gmail.com"),
		EmailSMTPPort:    getEnv("EMAIL_SMTP_PORT", "587"),
		EmailFrom:        getEnv("EMAIL_FROM", ""),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
