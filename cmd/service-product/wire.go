//go:build wireinject
// +build wireinject

// Package main provides wire dependency injection setup.
package main

import (
	"github.com/google/wire"
	"go.uber.org/zap"

	"github.com/ignata/go-microservices-boilerplate/internal/product/repository"
	"github.com/ignata/go-microservices-boilerplate/internal/product/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/config"
	"github.com/ignata/go-microservices-boilerplate/pkg/database"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
)

// InitializeApp creates a new application with all dependencies.
func InitializeApp(cfg *config.Config) (*App, error) {
	wire.Build(
		// Database
		providePostgresDB,
		provideRedisClient,

		// Logger
		provideLogger,

		// Event Bus
		provideEventBusProducer,

		// Repositories
		provideProductRepository,

		// Use Cases
		provideProductUseCase,

		// App
		NewApp,
	)
	return nil, nil
}

// providePostgresDB creates a PostgreSQL connection.
func providePostgresDB(cfg *config.Config) (*database.PostgresDB, error) {
	// Database should be created first using: make db-create
	// This just connects to the existing database
	return database.NewPostgresConnection(&database.PostgresConfig{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		Name:            cfg.Database.Name,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	})
}

// provideRedisClient creates a Redis client.
func provideRedisClient(cfg *config.Config) (*database.RedisClient, error) {
	return database.NewRedisConnection(&database.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
}

// provideLogger creates a logger.
func provideLogger(cfg *config.Config) (*zap.Logger, error) {
	return logger.New(&logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})
}

// provideEventBusProducer creates an event bus producer.
func provideEventBusProducer(redisClient *database.RedisClient, cfg *config.Config) *eventbus.Producer {
	return eventbus.NewProducer(redisClient.Client, eventbus.ProducerConfig{
		MaxLen:        cfg.Streams.MaxLen,
		DefaultSource: cfg.App.Name,
	})
}

// provideProductRepository creates a product repository.
func provideProductRepository(db *database.PostgresDB) repository.ProductRepository {
	return repository.NewProductRepository(db.DB)
}

// provideProductUseCase creates a product use case.
func provideProductUseCase(
	productRepo repository.ProductRepository,
	producer *eventbus.Producer,
	cfg *config.Config,
	logger *zap.Logger,
) usecase.ProductUseCase {
	return usecase.NewProductUseCase(
		productRepo,
		producer,
		usecase.Config{
			ServiceName: cfg.App.Name,
		},
		logger,
	)
}
