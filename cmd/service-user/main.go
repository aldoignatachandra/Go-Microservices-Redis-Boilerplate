// Package main provides the entry point for the user service.
//
// @title User Service API
// @version 1.0
// @description User service for Go Microservices Redis Pub/Sub Boilerplate. Manages user profiles, activity logs, and consumes auth events from Redis Streams.
//
// @contact.name API Support
// @contact.url https://github.com/aldoignatachandra/Go-Microservices-Redis-Boilerplate
//
// @host localhost:3101
// @BasePath /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}
package main

import (
	"context"
	"encoding/json"
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

	_ "github.com/ignata/go-microservices-boilerplate/cmd/service-user/docs"
	"github.com/ignata/go-microservices-boilerplate/internal/user/delivery"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/user/usecase"
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
	UserUseCase usecase.UserUseCase
	HTTPServer  *http.Server
}

// NewApp creates a new application.
func NewApp(
	cfg *config.Config,
	log *zap.Logger,
	pg *database.PostgresDB,
	redis *database.RedisClient,
	eventBus *eventbus.Producer,
	userUseCase usecase.UserUseCase,
) *App {
	return &App{
		Config:      cfg,
		Logger:      log,
		Postgres:    pg,
		Redis:       redis,
		EventBus:    eventBus,
		UserUseCase: userUseCase,
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

	// Start auth events consumer for activity logs
	consumerCtx, stopConsumer := context.WithCancel(context.Background())
	authEventConsumer := startAuthEventConsumer(consumerCtx, app)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting user service",
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

	// Stop consumer loop before closing dependencies
	stopConsumer()
	if authEventConsumer != nil {
		authEventConsumer.Stop()
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

	// Register user routes
	handler := delivery.NewUserHandler(app.UserUseCase)
	sessionValidator := buildSessionValidator(app)

	// Create Redis rate limiter if enabled
	if app.Config.RateLimit.Enabled && app.Redis != nil {
		redisLimiter := ratelimit.NewRedisRateLimiter(app.Redis, "ratelimit")
		redisLimiter.SetLimits(map[string]ratelimit.RouteLimit{
			"/api/v1/users":     {MaxRequests: 120, WindowSeconds: 60},
			"/api/v1/users/:id": {MaxRequests: 5, WindowSeconds: 60},
		})
		delivery.RegisterRoutesWithRateLimit(
			engine,
			handler,
			app.Config.Auth.JWT.Secret,
			sessionValidator,
			redisLimiter,
			app.Config.RateLimit.Requests,
			app.Config.RateLimit.Duration,
		)
	} else {
		delivery.RegisterRoutes(engine, handler, app.Config.Auth.JWT.Secret, sessionValidator)
	}

	return engine
}

func buildSessionValidator(app *App) pkgmiddleware.SessionValidator {
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

func startAuthEventConsumer(ctx context.Context, app *App) *eventbus.Consumer {
	consumerGroup, consumerName := resolveConsumerIdentity(app)

	consumer := eventbus.NewConsumer(app.Redis.Client, eventbus.ConsumerConfig{
		Stream:     eventbus.StreamAuthEvents,
		Group:      consumerGroup,
		Consumer:   consumerName,
		BatchSize:  app.Config.Streams.BatchSize,
		BlockMs:    app.Config.Streams.BlockMs,
		MaxRetries: 3,
	})

	go func() {
		app.Logger.Info("Starting auth events consumer",
			zap.String("stream", eventbus.StreamAuthEvents),
			zap.String("group", consumerGroup),
			zap.String("consumer", consumerName),
		)

		err := consumer.Consume(ctx, func(handlerCtx context.Context, event *eventbus.Event) error {
			actorUserID := resolveEventActorUserID(event)
			targetUserID, _ := event.Payload["user_id"].(string)
			if actorUserID == "" {
				app.Logger.Warn("Skipping auth event without user_id",
					zap.String("event_type", event.Type),
					zap.String("event_id", event.ID),
				)
				return nil
			}

			details, marshalErr := json.Marshal(map[string]interface{}{
				"event_id":   event.ID,
				"event_type": event.Type,
				"source":     event.Source,
				"payload":    event.Payload,
				"metadata":   event.Metadata,
			})
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal auth event details: %w", marshalErr)
			}

			ipAddress, userAgent := extractRequestInfo(event)

			if err := app.UserUseCase.LogActivity(handlerCtx, &dto.LogActivityRequest{
				UserID:    actorUserID,
				Action:    event.Type,
				Resource:  "auth",
				IPAddress: ipAddress,
				UserAgent: userAgent,
				Details:   string(details),
			}); err != nil {
				return fmt.Errorf("failed to create activity log: %w", err)
			}

			app.Logger.Info("Consumed auth event",
				zap.String("stream", eventbus.StreamAuthEvents),
				zap.String("event_id", event.ID),
				zap.String("event_type", event.Type),
				zap.String("actor_user_id", actorUserID),
				zap.String("target_user_id", targetUserID),
			)
			return nil
		}, func(_ context.Context, event *eventbus.Event, err error) {
			eventType := "unknown"
			eventID := ""
			if event != nil {
				eventType = event.Type
				eventID = event.ID
			}

			app.Logger.Error("Auth events consumer error",
				zap.String("stream", eventbus.StreamAuthEvents),
				zap.String("event_id", eventID),
				zap.String("event_type", eventType),
				zap.Error(err),
			)
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			app.Logger.Error("Auth events consumer stopped with error", zap.Error(err))
			return
		}

		app.Logger.Info("Auth events consumer stopped")
	}()

	return consumer
}

func extractRequestInfo(event *eventbus.Event) (string, string) {
	if event == nil || event.Payload == nil {
		return "", ""
	}

	ipAddress, _ := event.Payload["ip_address"].(string)
	userAgent, _ := event.Payload["user_agent"].(string)

	if strings.TrimSpace(ipAddress) == "" {
		ipAddress = extractMetadataString(event.Payload, "ip_address")
	}
	if strings.TrimSpace(userAgent) == "" {
		userAgent = extractMetadataString(event.Payload, "user_agent")
	}

	if strings.TrimSpace(ipAddress) == "" && event.Metadata != nil {
		ipAddress = event.Metadata["ip_address"]
	}
	if strings.TrimSpace(userAgent) == "" && event.Metadata != nil {
		userAgent = event.Metadata["user_agent"]
	}

	return strings.TrimSpace(ipAddress), strings.TrimSpace(userAgent)
}

func extractMetadataString(payload map[string]interface{}, key string) string {
	metadataRaw, ok := payload["metadata"]
	if !ok {
		return ""
	}

	switch metadata := metadataRaw.(type) {
	case map[string]interface{}:
		value, _ := metadata[key].(string)
		return strings.TrimSpace(value)
	case map[string]string:
		return strings.TrimSpace(metadata[key])
	default:
		return ""
	}
}

func resolveEventActorUserID(event *eventbus.Event) string {
	if event == nil || event.Payload == nil {
		return ""
	}

	if actorUserID, ok := event.Payload["actor_user_id"].(string); ok && strings.TrimSpace(actorUserID) != "" {
		return strings.TrimSpace(actorUserID)
	}

	if metadataRaw, ok := event.Payload["metadata"]; ok {
		switch metadata := metadataRaw.(type) {
		case map[string]interface{}:
			if value, ok := metadata["actor_user_id"].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		case map[string]string:
			if value := strings.TrimSpace(metadata["actor_user_id"]); value != "" {
				return value
			}
		}
	}

	if event.Metadata != nil {
		if value := strings.TrimSpace(event.Metadata["actor_user_id"]); value != "" {
			return value
		}
	}

	fallbackUserID, _ := event.Payload["user_id"].(string)
	return strings.TrimSpace(fallbackUserID)
}

func resolveConsumerIdentity(app *App) (string, string) {
	configuredGroup := strings.TrimSpace(app.Config.Streams.ConsumerGroup)
	configuredName := strings.TrimSpace(app.Config.Streams.ConsumerName)

	defaultGroup := app.Config.App.Name
	defaultName := fmt.Sprintf("%s-1", strings.TrimPrefix(app.Config.App.Name, "service-"))

	if defaultName == "-1" {
		defaultName = fmt.Sprintf("%s-1", app.Config.App.Name)
	}

	consumerGroup := configuredGroup
	if consumerGroup == "" {
		consumerGroup = defaultGroup
	}

	consumerName := configuredName
	if consumerName == "" {
		consumerName = defaultName
	}

	// Guard against shared auth identity in non-auth service.
	// Shared groups cause XREADGROUP load-balancing, which makes activity logs appear delayed/missing.
	if app.Config.App.Name != "service-auth" {
		if consumerGroup == "service-auth" {
			app.Logger.Warn("Detected shared auth consumer group in non-auth service; overriding to service-specific group",
				zap.String("configured_group", configuredGroup),
				zap.String("effective_group", defaultGroup),
			)
			consumerGroup = defaultGroup
		}
		if consumerName == "auth-1" {
			app.Logger.Warn("Detected shared auth consumer name in non-auth service; overriding to service-specific consumer",
				zap.String("configured_consumer", configuredName),
				zap.String("effective_consumer", defaultName),
			)
			consumerName = defaultName
		}
	}

	return consumerGroup, consumerName
}
