//go:build integration

package integration

import (
	"os"
)

// Получает переменную окружения или возвращает дефолт
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
