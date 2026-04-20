package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	OrderServiceAddr string        `envconfig:"ORDER_SERVICE_ADDR" default:"localhost:50052"`
	UserServiceAddr  string        `envconfig:"USER_SERVICE_ADDR" default:"localhost:50053"`
	TokenStoragePath string        `envconfig:"TOKEN_STORAGE_PATH" default:"tokens.json"`
	LogLevel         string        `envconfig:"LOG_LEVEL" default:"info"`
	RequestTimeout   time.Duration `envconfig:"REQUEST_TIMEOUT" default:"30s"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("envconfig failed: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.OrderServiceAddr == "" || c.UserServiceAddr == "" {
		return fmt.Errorf("service addresses are required")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("REQUEST_TIMEOUT must be positive")
	}
	return nil
}
