package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServiceName    string `envconfig:"SERVICE_NAME" default:"exchange-service"`
	ServiceVersion string `envconfig:"SERVICE_VERSION" default:"1.0.0"`
	LogLevel       string `envconfig:"LOG_LEVEL" default:"info"`

	GRPCPort     string        `envconfig:"GRPC_PORT" default:":50051"`
	GRPCTimeout  time.Duration `envconfig:"GRPC_TIMEOUT" default:"30s"`
	ReadTimeout  time.Duration `envconfig:"GRPC_READ_TIMEOUT" default:"10s"`
	WriteTimeout time.Duration `envconfig:"GRPC_WRITE_TIMEOUT" default:"10s"`

	HTTPPort string `envconfig:"HTTP_PORT" default:":8080"`

	DBHost     string        `envconfig:"DB_HOST" default:"localhost"`
	DBPort     int           `envconfig:"DB_PORT" default:"5432"`
	DBUser     string        `envconfig:"DB_USER" default:"postgres"`
	DBPassword string        `envconfig:"DB_PASSWORD" default:"postgres"`
	DBName     string        `envconfig:"DB_NAME" default:"exchange"`
	DBSSLMode  string        `envconfig:"DB_SSLMODE" default:"disable"`
	DBMaxConns int           `envconfig:"DB_MAX_CONNS" default:"25"`
	DBTimeout  time.Duration `envconfig:"DB_TIMEOUT" default:"5s"`

	RedisAddr     string        `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	RedisPassword string        `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB       int           `envconfig:"REDIS_DB" default:"0"`
	RedisPoolSize int           `envconfig:"REDIS_POOL_SIZE" default:"10"`
	RedisTimeout  time.Duration `envconfig:"REDIS_TIMEOUT" default:"5s"`

	JWTSecretOrPublicKey string        `envconfig:"JWT_SECRET_OR_PUBLIC_KEY" required:"true"`
	JWTIssuer            string        `envconfig:"JWT_ISSUER" default:"exchange-system"`
	JWTAudience          string        `envconfig:"JWT_AUDIENCE" default:"exchange-api"`
	JWTAlgorithm         string        `envconfig:"JWT_ALGORITHM" default:"HS256"`
	JWTClockSkew         time.Duration `envconfig:"JWT_CLOCK_SKEW" default:"30s"`

	AuthUseHttpOnlyCookies bool   `envconfig:"AUTH_USE_HTTP_ONLY_COOKIES" default:"false"`
	AuthCookieName         string `envconfig:"AUTH_COOKIE_NAME" default:"refresh_token"`
	AuthCookieDomain       string `envconfig:"AUTH_COOKIE_DOMAIN" default:""`
	AuthCookieSecure       bool   `envconfig:"AUTH_COOKIE_SECURE" default:"true"`
	AuthCookiePath         string `envconfig:"AUTH_COOKIE_PATH" default:"/"`

	IdempotencyKeyTTL          time.Duration `envconfig:"IDEMPOTENCY_KEY_TTL" default:"24h"`
	IdempotencyCleanupInterval time.Duration `envconfig:"IDEMPOTENCY_CLEANUP_INTERVAL" default:"1h"`
	IdempotencyMaxKeys         int           `envconfig:"IDEMPOTENCY_MAX_KEYS" default:"100000"`

	LogFormat          string   `envconfig:"LOG_FORMAT" default:"json"`
	LogSensitiveFields []string `envconfig:"LOG_SENSITIVE_FIELDS" default:"password,token,secret,key,card,cvv,pin"`
	LogRedactValue     string   `envconfig:"LOG_REDACT_VALUE" default:"***REDACTED***"`
	LogCaller          bool     `envconfig:"LOG_CALLER" default:"false"`
	LogStacktrace      bool     `envconfig:"LOG_STACKTRACE" default:"false"`

	RateLimitEnabled  bool          `envconfig:"RATE_LIMIT_ENABLED" default:"true"`
	RateLimitRequests int           `envconfig:"RATE_LIMIT_REQUESTS" default:"100"`
	RateLimitWindow   time.Duration `envconfig:"RATE_LIMIT_WINDOW" default:"1m"`
	RateLimitBurst    int           `envconfig:"RATE_LIMIT_BURST" default:"10"`

	CircuitBreakerEnabled     bool          `envconfig:"CIRCUIT_BREAKER_ENABLED" default:"true"`
	CircuitBreakerMaxFailures int           `envconfig:"CIRCUIT_BREAKER_MAX_FAILURES" default:"5"`
	CircuitBreakerTimeout     time.Duration `envconfig:"CIRCUIT_BREAKER_TIMEOUT" default:"30s"`
	CircuitBreakerInterval    time.Duration `envconfig:"CIRCUIT_BREAKER_INTERVAL" default:"10s"`

	RetryMaxAttempts     int           `envconfig:"RETRY_MAX_ATTEMPTS" default:"3"`
	RetryInitialInterval time.Duration `envconfig:"RETRY_INITIAL_INTERVAL" default:"100ms"`
	RetryMaxInterval     time.Duration `envconfig:"RETRY_MAX_INTERVAL" default:"10s"`
	RetryMultiplier      float64       `envconfig:"RETRY_MULTIPLIER" default:"2.0"`
	RetryRandomization   float64       `envconfig:"RETRY_RANDOMIZATION" default:"0.5"`

	PrometheusPort string `envconfig:"PROMETHEUS_PORT" default:":9090"`

	UserServiceAddr  string `envconfig:"USER_SERVICE_ADDR" default:"localhost:50053"`
	OrderServiceAddr string `envconfig:"ORDER_SERVICE_ADDR" default:"localhost:50052"`
	SpotServiceAddr  string `envconfig:"SPOT_SERVICE_ADDR" default:"localhost:50054"`

	TokenStoragePath string `envconfig:"TOKEN_STORAGE_PATH" default:"./tokens.json"`
}

type JWTConfig struct {
	Issuer    string        `envconfig:"JWT_ISSUER" default:"exchange-system"`
	Audience  string        `envconfig:"JWT_AUDIENCE" default:"exchange-api"`
	Algorithm string        `envconfig:"JWT_ALGORITHM" default:"RS256"`
	ClockSkew time.Duration `envconfig:"JWT_CLOCK_SKEW" default:"30s"`
}

func Load() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}
