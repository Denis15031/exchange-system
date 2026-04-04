package logger

import (
	"context"
	"fmt"
	"regexp"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/metadata"
)

type Config struct {
	Level           string
	Format          string
	SensitiveFields []string
	RedactValue     string
}

func DefaultConfig() Config {
	return Config{
		Level:           "info",
		Format:          "json",
		SensitiveFields: []string{"password", "token", "secret", "key", "card", "cvv", "pin", "idempotency_key"},
		RedactValue:     "***REDACTED***",
	}
}

type Logger struct {
	*zap.Logger
	patterns  []*regexp.Regexp
	redactVal string
}

func New(cfg Config) (*Logger, error) {
	zapLevel, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		zapLevel = zap.InfoLevel
	}

	patterns := make([]*regexp.Regexp, 0, len(cfg.SensitiveFields))
	for _, field := range cfg.SensitiveFields {
		pattern, err := regexp.Compile(`(?i)(` + regexp.QuoteMeta(field) + `)\s*[:=]\s*["']?([^"'\s,}]+)["']?`)
		if err == nil {
			patterns = append(patterns, pattern)
		}
	}

	zapCfg := zap.NewProductionConfig()
	if cfg.Format == "console" {
		zapCfg.Encoding = "console"
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	zapCfg.Level = zap.NewAtomicLevelAt(zapLevel)
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &Logger{
		Logger:    logger,
		patterns:  patterns,
		redactVal: cfg.RedactValue,
	}, nil
}

func (l *Logger) WithContext(ctx context.Context) *zap.Logger {
	fields := make([]zap.Field, 0, 2)

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if reqIDs := md.Get("x-request-id"); len(reqIDs) > 0 {
			fields = append(fields, zap.String("request_id", reqIDs[0]))
		}
		if userIDs := md.Get("x-user-id"); len(userIDs) > 0 {
			fields = append(fields, zap.String("user_id", userIDs[0]))
		}
	}

	return l.Logger.With(fields...)
}

func (l *Logger) Redact(s string) string {
	if len(l.patterns) == 0 {
		return s
	}
	result := s
	for _, pattern := range l.patterns {
		result = pattern.ReplaceAllString(result, "${1}="+l.redactVal)
	}
	return result
}

func (l *Logger) InfoRedact(msg string, fields ...zap.Field) {
	l.Logger.Info(l.Redact(msg), fields...)
}

func (l *Logger) ErrorRedact(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	l.Logger.Error(l.Redact(msg), fields...)
}

func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

func (l *Logger) Zap() *zap.Logger {
	return l.Logger
}
