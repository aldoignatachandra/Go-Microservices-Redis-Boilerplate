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

	_ "github.com/ignata/go-microservices-boilerplate/cmd/user-service/docs"
	"github.com/ignata/go-microservices-boilerplate/internal/user/delivery"
	"github.com/ignata/go-microservices-boilerplate/pkg/config"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"github.com/ignata/go-microservices-boilerplate/pkg/metrics"
	"github.com/ignata/go-microservices-boilerplate/pkg/server"
)

func main() {
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

	// Initialize application using Wire
	app, err := initializeApp(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize app", zap.Error(err))
	}

	// Setup HTTP server with proper middleware ordering
	setupHTTPServer(app, cfg)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      app.Engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting user service",
			zap.String("address", addr),
			zap.String("env", cfg.App.Env),
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

	// Create shutdown context with timeout from config
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	// Graceful cleanup of all resources
	if err := app.Shutdown(ctx); err != nil {
		logger.Error("Application shutdown error", zap.Error(err))
	}

	logger.Info("Server stopped")
}

// setupHTTPServer configures the HTTP server with all routes and middleware.
func setupHTTPServer(app *AppServer, cfg *config.Config) {
	// Set Gin mode based on environment
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Apply middleware in correct order:
	// 1. Recovery (catch panics first)
	// 2. CORS (handle preflight requests)
	// 3. Metrics (record all requests)
	app.Engine.Use(gin.Recovery())
	app.Engine.Use(delivery.CORSMiddleware())
	if cfg.Metrics.Enabled {
		app.Engine.Use(metrics.MetricsMiddleware(cfg.App.Name))
	}

	// Health check handler
	healthHandler := server.NewHealthHandler(server.HealthHandlerConfig{
		ServiceName: cfg.App.Name,
		Version:     cfg.App.Version,
		DB:          app.PostgresDB,
		Redis:       app.RedisClient,
	})

	// Register health routes
	app.Engine.GET("/health", healthHandler.PublicHealth)
	app.Engine.GET("/ready", healthHandler.ReadyProbe)
	app.Engine.GET("/live", healthHandler.LiveProbe)
	app.Engine.GET("/started", healthHandler.StartupProbe)

	// Metrics endpoint
	if cfg.Metrics.Enabled {
		app.Engine.GET(cfg.Metrics.Path, metrics.PrometheusHandler())
	}

	// Register user routes
	handler := delivery.NewUserHandler(app.UserUseCase)
	// Swagger endpoint
	app.Engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	delivery.RegisterRoutes(app.Engine, handler)
}
