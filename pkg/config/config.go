// Package config provides configuration management using Viper.
// It supports YAML files, environment variables, and follows 12-factor app principles.
package config

import (
	"time"
)

// Config holds all configuration for a microservice.
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Streams  StreamsConfig  `mapstructure:"streams"`
	Auth     AuthConfig     `mapstructure:"auth"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Tracing  TracingConfig  `mapstructure:"tracing"`
	Services ServicesConfig `mapstructure:"services"`
}

// AppConfig holds application-level configuration.
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig holds PostgreSQL database configuration.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Name            string        `mapstructure:"name"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// StreamsConfig holds Redis Streams configuration.
type StreamsConfig struct {
	MaxLen        int64  `mapstructure:"max_len"`
	BlockMs       int64  `mapstructure:"block_ms"`
	BatchSize     int64  `mapstructure:"batch_size"`
	ConsumerGroup string `mapstructure:"consumer_group"`
	ConsumerName  string `mapstructure:"consumer_name"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	JWT    JWTConfig    `mapstructure:"jwt"`
	Bcrypt BcryptConfig `mapstructure:"bcrypt"`
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret           string        `mapstructure:"secret"`
	ExpiresIn        time.Duration `mapstructure:"expires_in"`
	RefreshExpiresIn time.Duration `mapstructure:"refresh_expires_in"`
}

// BcryptConfig holds bcrypt configuration.
type BcryptConfig struct {
	Cost int `mapstructure:"cost"`
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Requests int           `mapstructure:"requests"`
	Duration time.Duration `mapstructure:"duration"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// MetricsConfig holds Prometheus metrics configuration.
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// TracingConfig holds OpenTelemetry tracing configuration.
type TracingConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
}

// ServicesConfig holds inter-service communication URLs.
type ServicesConfig struct {
	UserService    string `mapstructure:"user_service"`
	ProductService string `mapstructure:"product_service"`
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + itoa(c.Port) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Name +
		" sslmode=" + c.SSLMode
}

// Addr returns the Redis address in host:port format.
func (c *RedisConfig) Addr() string {
	return c.Host + ":" + itoa(c.Port)
}

// itoa converts int to string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var negative bool
	if i < 0 {
		negative = true
		i = -i
	}

	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}
