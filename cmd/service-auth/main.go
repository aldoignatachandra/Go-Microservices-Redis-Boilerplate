// Package main provides the entry point for the auth service.
//
// @title Auth Service API
// @version 1.0
// @description Authentication service for Go Microservices Redis Pub/Sub Boilerplate. Provides user registration, login, JWT token management, and admin user operations.
//
// @contact.name API Support
// @contact.url https://github.com/aldoignatachandra/Go-Microservices-Redis-Boilerplate
//
// @host localhost:3100
// @BasePath /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/ignata/go-microservices-boilerplate/cmd/service-auth/docs"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/delivery"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/config"
	"github.com/ignata/go-microservices-boilerplate/pkg/database"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"github.com/ignata/go-microservices-boilerplate/pkg/metrics"
	pkgmiddleware "github.com/ignata/go-microservices-boilerplate/pkg/middleware"
	"github.com/ignata/go-microservices-boilerplate/pkg/ratelimit"
	"github.com/ignata/go-microservices-boilerplate/pkg/server"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// App holds all application dependencies.
type App struct {
	Config      *config.Config
	Logger      *zap.Logger
	Postgres    *database.PostgresDB
	Redis       *database.RedisClient
	EventBus    *eventbus.Producer
	AuthUseCase usecase.AuthUseCase
	HTTPServer  *http.Server
}

// NewApp creates a new application.
func NewApp(
	cfg *config.Config,
	log *zap.Logger,
	pg *database.PostgresDB,
	redis *database.RedisClient,
	eventBus *eventbus.Producer,
	authUseCase usecase.AuthUseCase,
) *App {
	return &App{
		Config:      cfg,
		Logger:      log,
		Postgres:    pg,
		Redis:       redis,
		EventBus:    eventBus,
		AuthUseCase: authUseCase,
	}
}

func main() {
	// Load .env file
	utils.LoadEnv()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	if err := logger.Init(&logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	}); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// Initialize application using wire
	app, err := InitializeApp(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize app", zap.Error(err))
	}

	// Setup HTTP server
	engine := setupHTTPServer(app)

	// Start stream consumers for observability and cross-service event handling
	consumerCtx, stopConsumers := context.WithCancel(context.Background())
	streamConsumers := []*eventbus.Consumer{
		startStreamConsumer(consumerCtx, app, eventbus.StreamAuthEvents, "auth"),
		startStreamConsumer(consumerCtx, app, eventbus.StreamUserEvents, "users"),
		startStreamConsumer(consumerCtx, app, eventbus.StreamProductEvents, "products"),
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting auth service",
			zap.String("host", cfg.Server.Host),
			zap.Int("port", cfg.Server.Port),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Stop consumers before closing dependencies
	stopConsumers()
	for _, consumer := range streamConsumers {
		if consumer != nil {
			consumer.Stop()
		}
	}

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	// Close database connection
	if err := app.Postgres.Close(); err != nil {
		logger.Error("Database close error", zap.Error(err))
	}

	// Close Redis connection
	if err := app.Redis.Close(); err != nil {
		logger.Error("Redis close error", zap.Error(err))
	}

	logger.Info("Server stopped")
}

// setupHTTPServer configures the HTTP server with all routes and middleware.
func setupHTTPServer(app *App) *gin.Engine {
	// Set Gin mode
	if app.Config.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	engine := gin.New()

	// Add request tracing and structured request logging
	engine.Use(pkgmiddleware.RequestID())
	engine.Use(pkgmiddleware.RequestContextMetadata())
	engine.Use(pkgmiddleware.Logging(pkgmiddleware.LoggingConfig{
		Logger: app.Logger,
	}))

	// Add recovery middleware
	engine.Use(gin.Recovery())

	// Add CORS middleware
	engine.Use(delivery.CORSMiddleware())

	// Add metrics middleware
	if app.Config.Metrics.Enabled {
		engine.Use(metrics.MetricsMiddleware(app.Config.App.Name))
	}

	// Health check handler
	healthHandler := server.NewHealthHandler(server.HealthHandlerConfig{
		ServiceName: app.Config.App.Name,
		Version:     app.Config.App.Version,
		DB:          app.Postgres,
		Redis:       app.Redis,
	})

	// Register health routes
	engine.GET("/health", healthHandler.PublicHealth)
	engine.GET("/ready", healthHandler.ReadyProbe)
	engine.GET("/live", healthHandler.LiveProbe)
	engine.GET("/started", healthHandler.StartupProbe)
	admin := engine.Group("/admin")
	admin.GET("/health", healthHandler.AdminHealth)

	// Metrics endpoint
	if app.Config.Metrics.Enabled {
		engine.GET(app.Config.Metrics.Path, metrics.PrometheusHandler())
	}

	// Swagger endpoint
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Create Redis rate limiter if enabled
	var redisLimiter *ratelimit.RouteRateLimiter
	sessionValidator := buildSessionValidator(app)
	if app.Config.RateLimit.Enabled && app.Redis != nil {
		redisLimiter = ratelimit.NewRedisRateLimiter(
			app.Redis,
			ratelimit.BuildKeyPrefix(app.Config.App.Env, app.Config.App.Name),
		)

		redisLimiter.SetLimits(map[string]ratelimit.RouteLimit{
			"/api/v1/auth/login":    {MaxRequests: 10, WindowSeconds: 60},
			"/api/v1/auth/logout":   {MaxRequests: 30, WindowSeconds: 60},
			"/api/v1/auth/register": {MaxRequests: 10, WindowSeconds: 60},
		})
	}

	// Register auth routes with rate limiting
	if redisLimiter != nil && app.Config.RateLimit.Enabled {
		delivery.RegisterRoutesWithRateLimit(
			engine,
			app.AuthUseCase,
			app.Config.Auth.JWT.Secret,
			sessionValidator,
			redisLimiter,
			app.Config.RateLimit.Requests,
			app.Config.RateLimit.Duration,
		)
	} else {
		delivery.RegisterRoutes(engine, app.AuthUseCase, app.Config.Auth.JWT.Secret, sessionValidator)
	}

	return engine
}

func buildSessionValidator(app *App) delivery.SessionValidator {
	if app == nil || app.Postgres == nil {
		return nil
	}

	return func(ctx context.Context, userID, sessionID string) (bool, error) {
		if strings.TrimSpace(sessionID) == "" {
			return false, nil
		}
		return app.Postgres.HasActiveSessionByID(ctx, userID, sessionID)
	}
}

func startStreamConsumer(ctx context.Context, app *App, stream, consumerSuffix string) *eventbus.Consumer {
	if app.Redis == nil {
		app.Logger.Warn("Redis client is nil, stream consumer not started",
			zap.String("stream", stream),
		)
		return nil
	}

	consumerGroup := app.Config.Streams.ConsumerGroup
	if consumerGroup == "" {
		consumerGroup = app.Config.App.Name
	}

	consumerName := app.Config.Streams.ConsumerName
	if consumerName == "" {
		consumerName = fmt.Sprintf("%s-1", app.Config.App.Name)
	}
	consumerName = fmt.Sprintf("%s-%s", consumerName, consumerSuffix)

	consumer := eventbus.NewConsumer(app.Redis.Client, eventbus.ConsumerConfig{
		Stream:     stream,
		Group:      consumerGroup,
		Consumer:   consumerName,
		BatchSize:  app.Config.Streams.BatchSize,
		BlockMs:    app.Config.Streams.BlockMs,
		MaxRetries: 3,
	})

	go func() {
		app.Logger.Info("Starting stream consumer",
			zap.String("stream", stream),
			zap.String("group", consumerGroup),
			zap.String("consumer", consumerName),
		)

		err := consumer.Consume(ctx, func(_ context.Context, event *eventbus.Event) error {
			app.Logger.Info("Consumed stream event",
				zap.String("stream", stream),
				zap.String("event_id", event.ID),
				zap.String("event_type", event.Type),
				zap.String("source", event.Source),
			)
			return nil
		}, func(_ context.Context, event *eventbus.Event, err error) {
			eventType := "unknown"
			eventID := ""
			if event != nil {
				eventType = event.Type
				eventID = event.ID
			}
			app.Logger.Error("Stream consumer error",
				zap.String("stream", stream),
				zap.String("event_id", eventID),
				zap.String("event_type", eventType),
				zap.Error(err),
			)
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			app.Logger.Error("Stream consumer stopped with error",
				zap.String("stream", stream),
				zap.Error(err),
			)
			return
		}

		app.Logger.Info("Stream consumer stopped", zap.String("stream", stream))
	}()

	return consumer
}
