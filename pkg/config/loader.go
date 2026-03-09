// Package config provides configuration loading utilities.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Load loads configuration from YAML file and environment variables.
// It follows the 12-factor app methodology:
// 1. Load from YAML file (configs/{env}.yaml)
// 2. Override with environment variables
// 3. Environment variables take precedence
//
// Environment variables should be prefixed with the app name or use uppercase:
// - APP_NAME -> app.name
// - DB_HOST -> database.host
// - REDIS_HOST -> redis.host
func Load(configPath string) (*Config, error) {
	return LoadWithEnv(configPath, "")
}

// LoadWithEnv loads configuration for a specific environment.
// env can be: local, development, staging, production
func LoadWithEnv(configPath string, env string) (*Config, error) {
	v := viper.New()

	// Set default config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Default config path
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")

		// Determine config file from environment
		if env == "" {
			env = getEnv("APP_ENV", "local")
		}
		v.SetConfigName(env)
		v.SetConfigType("yaml")
	}

	// Set default values
	setDefaults(v)

	// Enable environment variable override
	v.SetEnvPrefix("") // No prefix, allows APP_NAME, DB_HOST, etc.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind specific env vars for explicit mapping
	bindEnvVars(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults and env vars
	}

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Post-process configuration
	postProcess(&cfg)

	return &cfg, nil
}

// LoadFromEnv loads configuration primarily from environment variables.
// This is useful for containerized environments.
func LoadFromEnv() (*Config, error) {
	v := viper.New()
	setDefaults(v)

	// Enable environment variable override
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind specific env vars that might not match the pattern
	bindEnvVars(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	postProcess(&cfg)

	return &cfg, nil
}

// setDefaults sets default values for all configuration options.
func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("app.name", "service")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.env", "local")

	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 3100)
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.shutdown_timeout", "10s")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)

	// Streams defaults
	v.SetDefault("streams.max_len", 10000)
	v.SetDefault("streams.block_ms", 5000)
	v.SetDefault("streams.batch_size", 10)

	// Auth defaults
	v.SetDefault("auth.jwt.expires_in", "24h")
	v.SetDefault("auth.jwt.refresh_expires_in", "168h")
	v.SetDefault("auth.bcrypt.cost", 12)

	// Rate limit defaults
	v.SetDefault("rate_limit.enabled", true)
	v.SetDefault("rate_limit.requests", 100)
	v.SetDefault("rate_limit.duration", "1m")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")

	// Tracing defaults
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.endpoint", "http://localhost:4317")

	// Services defaults
	v.SetDefault("services.user_service", "http://localhost:3101")
	v.SetDefault("services.product_service", "http://localhost:3102")
}

// bindEnvVars binds specific environment variables to config keys.
// This handles cases where the automatic mapping doesn't work.
func bindEnvVars(v *viper.Viper) {
	// App
	_ = v.BindEnv("app.name", "APP_NAME")
	_ = v.BindEnv("app.version", "APP_VERSION")
	_ = v.BindEnv("app.env", "APP_ENV")

	// Server
	_ = v.BindEnv("server.host", "SERVER_HOST")
	_ = v.BindEnv("server.port", "SERVER_PORT")
	_ = v.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	_ = v.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	_ = v.BindEnv("server.shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT")

	// Database
	_ = v.BindEnv("database.host", "DB_HOST")
	_ = v.BindEnv("database.port", "DB_PORT")
	_ = v.BindEnv("database.name", "DB_NAME")
	_ = v.BindEnv("database.user", "DB_USER")
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("database.sslmode", "DB_SSLMODE")
	_ = v.BindEnv("database.max_open_conns", "DB_MAX_OPEN_CONNS")
	_ = v.BindEnv("database.max_idle_conns", "DB_MAX_IDLE_CONNS")
	_ = v.BindEnv("database.conn_max_lifetime", "DB_CONN_MAX_LIFETIME")

	// Redis
	_ = v.BindEnv("redis.host", "REDIS_HOST")
	_ = v.BindEnv("redis.port", "REDIS_PORT")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("redis.db", "REDIS_DB")
	_ = v.BindEnv("redis.pool_size", "REDIS_POOL_SIZE")

	// Auth
	_ = v.BindEnv("auth.jwt.secret", "JWT_SECRET")
	_ = v.BindEnv("auth.jwt.expires_in", "JWT_EXPIRES_IN")
	_ = v.BindEnv("auth.jwt.refresh_expires_in", "JWT_REFRESH_EXPIRES_IN")
	_ = v.BindEnv("auth.bcrypt.cost", "BCRYPT_COST")

	// Logging
	_ = v.BindEnv("logging.level", "LOG_LEVEL")
	_ = v.BindEnv("logging.format", "LOG_FORMAT")
}

// postProcess performs post-processing on the configuration.
func postProcess(cfg *Config) {
	// Parse duration strings if they weren't parsed correctly
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 10 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 10 * time.Second
	}
	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = 10 * time.Second
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = 5 * time.Minute
	}
	if cfg.Auth.JWT.ExpiresIn == 0 {
		cfg.Auth.JWT.ExpiresIn = 24 * time.Hour
	}
	if cfg.Auth.JWT.RefreshExpiresIn == 0 {
		cfg.Auth.JWT.RefreshExpiresIn = 168 * time.Hour
	}
	if cfg.RateLimit.Duration == 0 {
		cfg.RateLimit.Duration = time.Minute
	}
}

// getEnv gets an environment variable or returns the default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// MustLoad loads configuration and panics on error.
// Use this in main when you want to fail fast on config errors.
func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// MustLoadFromEnv loads configuration from environment and panics on error.
func MustLoadFromEnv() *Config {
	cfg, err := LoadFromEnv()
	if err != nil {
		panic(fmt.Sprintf("failed to load config from env: %v", err))
	}
	return cfg
}
