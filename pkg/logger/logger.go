// Package logger provides structured logging using Uber's Zap.
// It supports both development (console) and production (JSON) formats.
package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// defaultLogLevel is the default logging level.
	defaultLogLevel = "info"
)

// Logger is an alias for zap.Logger for convenience.
type Logger = *zap.Logger

var (
	// Global logger instance
	globalLogger *zap.Logger
	// Global sugar logger (for convenience)
	globalSugar *zap.SugaredLogger
	// Ensure thread-safe initialization
	once sync.Once
)

// Config holds logger configuration.
type Config struct {
	// Level: debug, info, warn, error
	Level string `mapstructure:"level"`
	// Format: console (development) or json (production)
	Format string `mapstructure:"format"`
}

// New creates a new zap.Logger based on the provided configuration.
// If config is nil, defaults to debug level with console format.
func New(cfg *Config) (*zap.Logger, error) {
	var zapCfg zap.Config

	// Determine format: console for development, JSON for production
	if cfg == nil || cfg.Format == "console" {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapCfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	} else {
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Set log level
	if cfg != nil {
		switch cfg.Level {
		case "debug":
			zapCfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		case defaultLogLevel:
			zapCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		case "warn":
			zapCfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
		case "error":
			zapCfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
		default:
			zapCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		}
	}

	// Build logger with caller info and stacktrace for errors
	logger, err := zapCfg.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// Init initializes the global logger.
// This should be called once at application startup.
func Init(cfg *Config) error {
	var initErr error
	once.Do(func() {
		logger, err := New(cfg)
		if err != nil {
			initErr = err
			return
		}
		globalLogger = logger
		globalSugar = logger.Sugar()
	})
	return initErr
}

// L returns the global logger instance.
// Falls back to a no-op logger if not initialized.
func L() *zap.Logger {
	if globalLogger == nil {
		return zap.NewNop()
	}
	return globalLogger
}

// S returns the global sugared logger instance.
// Falls back to a no-op logger if not initialized.
func S() *zap.SugaredLogger {
	if globalSugar == nil {
		return zap.NewNop().Sugar()
	}
	return globalSugar
}

// Sync flushes any buffered log entries.
// Should be called before application exit.
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// Debug logs a message at debug level.
func Debug(msg string, fields ...zap.Field) {
	L().Debug(msg, fields...)
}

// Info logs a message at info level.
func Info(msg string, fields ...zap.Field) {
	L().Info(msg, fields...)
}

// Warn logs a message at warn level.
func Warn(msg string, fields ...zap.Field) {
	L().Warn(msg, fields...)
}

// Error logs a message at error level.
func Error(msg string, fields ...zap.Field) {
	L().Error(msg, fields...)
}

// Fatal logs a message at fatal level and exits.
func Fatal(msg string, fields ...zap.Field) {
	L().Fatal(msg, fields...)
}

// Panic logs a message at panic level and panics.
func Panic(msg string, fields ...zap.Field) {
	L().Panic(msg, fields...)
}

// With creates a child logger with additional fields.
func With(fields ...zap.Field) *zap.Logger {
	return L().With(fields...)
}

// Named adds a sub-logger name.
func Named(name string) *zap.Logger {
	return L().Named(name)
}

// SetLevel changes the global logger level at runtime.
func SetLevel(level string) {
	// Validate the level
	switch level {
	case "debug", "info", "warn", "error":
		// Valid level
	default:
		level = "info"
	}

	// Note: This only works if the logger was created with an AtomicLevel
	// For now, we'll recreate the logger
	cfg := &Config{Level: level, Format: "json"}
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
	_ = Init(cfg)
}

// GetEnvOrDefault returns environment variable or default value.
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
