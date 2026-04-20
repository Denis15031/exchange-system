package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServiceName      string `envconfig:"SERVICE_NAME" default:"spot-service"`
	GRPCPort         string `envconfig:"GRPC_PORT" default:":50054"`
	MetricsPort      string `envconfig:"METRICS_PORT" default:":9094"`
	LogLevel         string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat        string `envconfig:"LOG_FORMAT" default:"json"`
	JWTPublicKeyPath string `envconfig:"JWT_PUBLIC_KEY_PATH" default:"./keys/public.pem"`

	RateLimitRPS   int           `envconfig:"RATE_LIMIT_RPS" default:"100"`
	RateLimitBurst int           `envconfig:"RATE_LIMIT_BURST" default:"50"`
	IdempotencyTTL time.Duration `envconfig:"IDEMPOTENCY_TTL" default:"24h"`

	ShutdownGracePeriod    time.Duration `envconfig:"SHUTDOWN_GRACE_PERIOD" default:"30s"`
	MetricsShutdownTimeout time.Duration `envconfig:"METRICS_SHUTDOWN_TIMEOUT" default:"5s"`
	MetricsReadTimeout     time.Duration `envconfig:"METRICS_READ_TIMEOUT" default:"5s"`
	MetricsWriteTimeout    time.Duration `envconfig:"METRICS_WRITE_TIMEOUT" default:"10s"`
	MetricsIdleTimeout     time.Duration `envconfig:"METRICS_IDLE_TIMEOUT" default:"120s"`

	GRPCKeepaliveMaxConnIdle time.Duration `envconfig:"GRPC_KEEPALIVE_MAX_CONN_IDLE" default:"5m"`
	GRPCKeepaliveTime        time.Duration `envconfig:"GRPC_KEEPALIVE_TIME" default:"2m"`
	GRPCKeepaliveTimeout     time.Duration `envconfig:"GRPC_KEEPALIVE_TIMEOUT" default:"20s"`

	UserServiceAddr  string `envconfig:"USER_SERVICE_ADDR" default:"localhost:50053"`
	OrderServiceAddr string `envconfig:"ORDER_SERVICE_ADDR" default:"localhost:50052"`

	MarketRefreshInterval time.Duration `envconfig:"MARKET_REFRESH_INTERVAL" default:"5s"`
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
	if c.GRPCPort == "" || c.UserServiceAddr == "" || c.OrderServiceAddr == "" {
		return fmt.Errorf("required service addresses missing")
	}
	if c.RateLimitRPS <= 0 || c.ShutdownGracePeriod <= 0 {
		return fmt.Errorf("invalid rate limit or timeout values")
	}
	return nil
}
