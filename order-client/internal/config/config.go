package config

import (
	"os"
	"time"
)

type Config struct {
	OrderServiceAddr string
	UserServiceAddr  string
	TokenStoragePath string
	LogLevel         string
	RequestTimeout   time.Duration
}

func Load() (*Config, error) {
	return &Config{
		OrderServiceAddr: getEnv("ORDER_SERVICE_ADDR", "localhost:50052"),
		UserServiceAddr:  getEnv("USER_SERVICE_ADDR", "localhost:50053"),
		TokenStoragePath: getEnv("TOKEN_STORAGE_PATH", "tokens.json"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		RequestTimeout:   30 * time.Second,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
