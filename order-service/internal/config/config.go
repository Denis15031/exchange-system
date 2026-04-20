package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServiceName string `envconfig:"SERVICE_NAME" default:"order-service"`
	GRPCPort    string `envconfig:"GRPC_PORT" default:":50052"`
	MetricsPort string `envconfig:"METRICS_PORT" default:":9092"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat   string `envconfig:"LOG_FORMAT" default:"json"`

	DBHost     string        `envconfig:"DB_HOST" default:"localhost"`
	DBPort     int           `envconfig:"DB_PORT" default:"5432"`
	DBUser     string        `envconfig:"DB_USER" default:"postgres"`
	DBPassword string        `envconfig:"DB_PASSWORD" default:"postgres"`
	DBName     string        `envconfig:"DB_NAME" default:"exchange"`
	DBSSLMode  string        `envconfig:"DB_SSLMODE" default:"disable"`
	DBMaxConns int           `envconfig:"DB_MAX_CONNS" default:"25"`
	DBTimeout  time.Duration `envconfig:"DB_TIMEOUT" default:"5s"`
	DBDSN      string        `envconfig:"DATABASE_URL" default:""`

	RedisAddr     string        `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	RedisPassword string        `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB       int           `envconfig:"REDIS_DB" default:"0"`
	RedisPoolSize int           `envconfig:"REDIS_POOL_SIZE" default:"10"`
	RedisTimeout  time.Duration `envconfig:"REDIS_TIMEOUT" default:"5s"`

	JWTSecretOrPublicKey string        `envconfig:"JWT_SECRET_OR_PUBLIC_KEY" default:""`
	JWTIssuer            string        `envconfig:"JWT_ISSUER" default:"exchange-system"`
	JWTAudience          string        `envconfig:"JWT_AUDIENCE" default:"exchange-api"`
	JWTAlgorithm         string        `envconfig:"JWT_ALGORITHM" default:"HS256"`
	JWTClockSkew         time.Duration `envconfig:"JWT_CLOCK_SKEW" default:"30s"`

	IdempotencyTTL             time.Duration `envconfig:"IDEMPOTENCY_TTL" default:"24h"`
	IdempotencyCleanupInterval time.Duration `envconfig:"IDEMPOTENCY_CLEANUP_INTERVAL" default:"1h"`
	IdempotencyMaxKeys         int           `envconfig:"IDEMPOTENCY_MAX_KEYS" default:"100000"`

	RateLimitRPS   int `envconfig:"RATE_LIMIT_RPS" default:"100"`
	RateLimitBurst int `envconfig:"RATE_LIMIT_BURST" default:"50"`

	GRPCKeepaliveMaxConnIdle time.Duration `envconfig:"GRPC_KEEPALIVE_MAX_CONN_IDLE" default:"5m"`
	GRPCKeepaliveTime        time.Duration `envconfig:"GRPC_KEEPALIVE_TIME" default:"2m"`
	GRPCKeepaliveTimeout     time.Duration `envconfig:"GRPC_KEEPALIVE_TIMEOUT" default:"20s"`

	ShutdownGracePeriod    time.Duration `envconfig:"SHUTDOWN_GRACE_PERIOD" default:"30s"`
	MetricsShutdownTimeout time.Duration `envconfig:"METRICS_SHUTDOWN_TIMEOUT" default:"5s"`
	MetricsReadTimeout     time.Duration `envconfig:"METRICS_READ_TIMEOUT" default:"5s"`
	MetricsWriteTimeout    time.Duration `envconfig:"METRICS_WRITE_TIMEOUT" default:"10s"`
	MetricsIdleTimeout     time.Duration `envconfig:"METRICS_IDLE_TIMEOUT" default:"120s"`
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
	if c.RateLimitRPS <= 0 {
		return fmt.Errorf("RATE_LIMIT_RPS must be positive")
	}
	if c.ShutdownGracePeriod <= 0 {
		return fmt.Errorf("SHUTDOWN_GRACE_PERIOD must be positive")
	}
	return nil
}
