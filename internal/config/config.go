package config

import (
	"os"
	"time"
)

// Config содержит настройки приложения, читаемые из переменных окружения.
type Config struct {
	Port            string
	ShutdownTimeout time.Duration
}

// Load читает конфигурацию из переменных окружения, подставляя разумные значения по умолчанию.
func Load() Config {
	return Config{
		Port:            getEnv("PORT", "8080"),
		ShutdownTimeout: 10 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
