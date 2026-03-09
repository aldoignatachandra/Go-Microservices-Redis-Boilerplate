// Package main provides the entry point for the product service.
//
// @title Product Service API
// @version 1.0
// @description Product service for Go Microservices Redis Pub/Sub Boilerplate. Manages product CRUD operations and stock management with Redis event publishing.
//
// @contact.name API Support
// @contact.url https://github.com/aldoignatachandra/Go-Microservices-Redis-Boilerplate
//
// @host localhost:3102
// @BasePath /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/ignata/go-microservices-boilerplate/cmd/service-product/docs"
	"github.com/ignata/go-microservices-boilerplate/internal/product/delivery"
	"github.com/ignata/go-microservices-boilerplate/internal/product/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/config"
	"github.com/ignata/go-microservices-boilerplate/pkg/database"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"github.com/ignata/go-microservices-boilerplate/pkg/metrics"
	"github.com/ignata/go-microservices-boilerplate/pkg/ratelimit"
	"github.com/ignata/go-microservices-boilerplate/pkg/server"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// App holds all application dependencies.
type App struct {
	Config         *config.Config
	Logger         *zap.Logger
	Postgres       *database.PostgresDB
	Redis          *database.RedisClient
	EventBus       *eventbus.Producer
	ProductUseCase usecase.ProductUseCase
	HTTPServer     *http.Server
}

// NewApp creates a new application.
func NewApp(
	cfg *config.Config,
	log *zap.Logger,
	pg *database.PostgresDB,
	redis *database.RedisClient,
	eventBus *eventbus.Producer,
	productUseCase usecase.ProductUseCase,
) *App {
	return &App{
		Config:         cfg,
		Logger:         log,
		Postgres:       pg,
		Redis:          redis,
		EventBus:       eventBus,
		ProductUseCase: productUseCase,
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

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting product service",
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

	// Metrics endpoint
	if app.Config.Metrics.Enabled {
		engine.GET(app.Config.Metrics.Path, metrics.PrometheusHandler())
	}

	// Swagger endpoint
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Register product routes with rate limiting
	if app.Config.RateLimit.Enabled && app.Redis != nil {
		redisLimiter := ratelimit.NewRedisRateLimiter(app.Redis, "ratelimit")
		redisLimiter.SetLimits(map[string]ratelimit.RouteLimit{
			"/products":           {MaxRequests: 120, WindowSeconds: 60},
			"/products/:id":       {MaxRequests: 10, WindowSeconds: 60},
			"/products/:id/stock": {MaxRequests: 30, WindowSeconds: 60},
		})
		delivery.RegisterRoutesWithRateLimit(engine, app.ProductUseCase, redisLimiter, app.Config.RateLimit.Requests, app.Config.RateLimit.Duration)
	} else {
		delivery.RegisterRoutes(engine, app.ProductUseCase)
	}

	return engine
}
