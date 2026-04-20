package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServiceName string `envconfig:"SERVICE_NAME" default:"user-service"`
	GRPCPort    string `envconfig:"GRPC_PORT" default:":50053"`
	MetricsPort string `envconfig:"METRICS_PORT" default:":9093"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat   string `envconfig:"LOG_FORMAT" default:"json"`

	JWTPrivateKeyPath string        `envconfig:"JWT_PRIVATE_KEY_PATH" default:"./keys/private.pem"`
	JWTPublicKeyPath  string        `envconfig:"JWT_PUBLIC_KEY_PATH" default:"./keys/public.pem"`
	JWTAccessTTL      time.Duration `envconfig:"JWT_ACCESS_TTL" default:"15m"`
	JWTRefreshTTL     time.Duration `envconfig:"JWT_REFRESH_TTL" default:"168h"`
	JWTIssuer         string        `envconfig:"JWT_ISSUER" default:"exchange-system"`
	JWTAudience       string        `envconfig:"JWT_AUDIENCE" default:"exchange-api"`
	JWTAlgorithm      string        `envconfig:"JWT_ALGORITHM" default:"RS256"`
	JWTClockSkew      time.Duration `envconfig:"JWT_CLOCK_SKEW" default:"30s"`

	BcryptCost int `envconfig:"BCRYPT_COST" default:"12"`

	RateLimitRPS   int `envconfig:"RATE_LIMIT_RPS" default:"100"`
	RateLimitBurst int `envconfig:"RATE_LIMIT_BURST" default:"50"`

	IdempotencyTTL time.Duration `envconfig:"IDEMPOTENCY_TTL" default:"24h"`

	AuthUseHttpOnlyCookies bool   `envconfig:"AUTH_USE_HTTP_ONLY_COOKIES" default:"false"`
	AuthCookieName         string `envconfig:"AUTH_COOKIE_NAME" default:"refresh_token"`
	AuthCookieDomain       string `envconfig:"AUTH_COOKIE_DOMAIN" default:""`
	AuthCookieSecure       bool   `envconfig:"AUTH_COOKIE_SECURE" default:"true"`
	AuthCookiePath         string `envconfig:"AUTH_COOKIE_PATH" default:"/"`

	ShutdownGracePeriod    time.Duration `envconfig:"SHUTDOWN_GRACE_PERIOD" default:"30s"`
	MetricsShutdownTimeout time.Duration `envconfig:"METRICS_SHUTDOWN_TIMEOUT" default:"5s"`

	GRPCKeepaliveMaxConnIdle time.Duration `envconfig:"GRPC_KEEPALIVE_MAX_CONN_IDLE" default:"5m"`
	GRPCKeepaliveTime        time.Duration `envconfig:"GRPC_KEEPALIVE_TIME" default:"2m"`
	GRPCKeepaliveTimeout     time.Duration `envconfig:"GRPC_KEEPALIVE_TIMEOUT" default:"20s"`

	MetricsReadTimeout  time.Duration `envconfig:"METRICS_READ_TIMEOUT" default:"5s"`
	MetricsWriteTimeout time.Duration `envconfig:"METRICS_WRITE_TIMEOUT" default:"10s"`
	MetricsIdleTimeout  time.Duration `envconfig:"METRICS_IDLE_TIMEOUT" default:"120s"`
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
	if c.BcryptCost < 10 || c.BcryptCost > 14 {
		return fmt.Errorf("BCRYPT_COST must be 10-14, got %d", c.BcryptCost)
	}
	if c.JWTAccessTTL <= 0 || c.JWTRefreshTTL <= 0 {
		return fmt.Errorf("JWT TTLs must be positive")
	}
	if c.RateLimitRPS <= 0 {
		return fmt.Errorf("RATE_LIMIT_RPS must be positive")
	}
	if c.ShutdownGracePeriod <= 0 {
		return fmt.Errorf("SHUTDOWN_GRACE_PERIOD must be positive")
	}
	return nil
}
