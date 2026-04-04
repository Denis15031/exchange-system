package config

import (
	"os"
	"time"
)

type Config struct {
	GRPCPort    string
	MetricsPort string
	LogLevel    string

	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
	JWTAccessTTL      time.Duration
	JWTRefreshTTL     time.Duration

	BcryptCost int

	LoginRateLimit int
}

func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:          getEnv("GRPC_PORT", ":50053"),
		MetricsPort:       getEnv("METRICS_PORT", ":9093"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		JWTPrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", ""),
		JWTPublicKeyPath:  getEnv("JWT_PUBLIC_KEY_PATH", ""),
		JWTAccessTTL:      15 * time.Minute,
		JWTRefreshTTL:     7 * 24 * time.Hour,
		BcryptCost:        12,
		LoginRateLimit:    10, // 10 попыток в минуту
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
