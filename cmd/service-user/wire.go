//go:build wireinject
// +build wireinject

// Package main provides Wire dependency injection setup for the user service.
package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"go.uber.org/zap"

	"github.com/ignata/go-microservices-boilerplate/internal/user/repository"
	"github.com/ignata/go-microservices-boilerplate/internal/user/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/config"
	"github.com/ignata/go-microservices-boilerplate/pkg/database"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"github.com/ignata/go-microservices-boilerplate/pkg/server"
)

// AppServer contains all dependencies for the user service.
type AppServer struct {
	Engine      *gin.Engine
	UserUseCase usecase.UserUseCase
	PostgresDB  *database.PostgresDB
	RedisClient *database.RedisClient
	Log         *zap.Logger
	Config      *config.Config
}

// Shutdown gracefully shuts down the server.
func (s *AppServer) Shutdown(ctx context.Context) error {
	return server.GracefulShutdown(ctx, s.Config, s.Engine, s.Log, s.PostgresDB.DB, s.RedisClient)
}

// initializeApp initializes the application with all dependencies using Wire.
//
//go:generate wire
func initializeApp(cfg *config.Config) (*AppServer, error) {
	wire.Build(
		// Core providers
		wire.Struct(new(AppServer), "*"),
		provideLogger,
		provideGinEngine,
		providePostgresDB,
		provideRedisClient,
		provideEventBusProducer,
		provideUserRepository,
		provideActivityRepository,
		provideUserUseCase,
	)
	return &AppServer{}, nil
}

// provideLogger creates a logger.
func provideLogger(cfg *config.Config) (*zap.Logger, error) {
	return logger.New(&logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})
}

// provideGinEngine creates a Gin engine.
func provideGinEngine() *gin.Engine {
	return gin.Default()
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

// provideEventBusProducer creates an event bus producer.
func provideEventBusProducer(redisClient *database.RedisClient, cfg *config.Config) *eventbus.Producer {
	return eventbus.NewProducer(redisClient.Client, eventbus.ProducerConfig{
		MaxLen:        cfg.Streams.MaxLen,
		DefaultSource: cfg.App.Name,
	})
}

// provideUserRepository creates a user repository.
func provideUserRepository(db *database.PostgresDB) repository.UserRepository {
	return repository.NewUserRepository(db.DB)
}

// provideActivityRepository creates an activity repository.
func provideActivityRepository(db *database.PostgresDB) repository.ActivityRepository {
	return repository.NewActivityRepository(db.DB)
}

// provideUserUseCase creates a user use case.
func provideUserUseCase(
	userRepo repository.UserRepository,
	activityRepo repository.ActivityRepository,
	producer *eventbus.Producer,
	log *zap.Logger,
) usecase.UserUseCase {
	return usecase.NewUserUseCase(
		userRepo,
		activityRepo,
		producer,
		log,
	)
}
