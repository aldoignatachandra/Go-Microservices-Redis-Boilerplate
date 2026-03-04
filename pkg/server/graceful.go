// Package server provides graceful shutdown utilities.
package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ignata/go-microservices-boilerplate/pkg/database"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GracefulServer wraps Server with graceful shutdown capabilities.
type GracefulServer struct {
	*Server
	shutdownTimeout time.Duration
	shutdownHooks   []func(context.Context) error
}

// NewGracefulServer creates a new server with graceful shutdown support.
func NewGracefulServer(cfg Config) *GracefulServer {
	return &GracefulServer{
		Server:          New(cfg),
		shutdownTimeout: cfg.ShutdownTimeout,
		shutdownHooks:   make([]func(context.Context) error, 0),
	}
}

// RegisterShutdownHook registers a function to be called during shutdown.
// Hooks are called in the order they were registered.
func (s *GracefulServer) RegisterShutdownHook(hook func(context.Context) error) {
	s.shutdownHooks = append(s.shutdownHooks, hook)
}

// WaitForShutdown blocks until a termination signal is received,
// then gracefully shuts down the server.
func (s *GracefulServer) WaitForShutdown() {
	// Create channel for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-quit
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	// Execute shutdown hooks first (in reverse order for proper cleanup)
	for i := len(s.shutdownHooks) - 1; i >= 0; i-- {
		hook := s.shutdownHooks[i]
		if err := hook(ctx); err != nil {
			log.Printf("Shutdown hook error: %v", err)
		}
	}

	// Shutdown HTTP server
	if err := s.Server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
		// Force close if shutdown times out
		_ = s.Server.Close()
	}

	log.Println("Server stopped")
}

// WaitForShutdownWithContext is like WaitForShutdown but accepts a context.
// This allows for external cancellation.
func (s *GracefulServer) WaitForShutdownWithContext(ctx context.Context) {
	// Create channel for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case <-ctx.Done():
		log.Println("Context cancelled, initiating graceful shutdown...")
	case sig := <-quit:
		log.Printf("Received signal %v, initiating graceful shutdown...", sig)
	}

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	// Execute shutdown hooks
	for i := len(s.shutdownHooks) - 1; i >= 0; i-- {
		hook := s.shutdownHooks[i]
		if err := hook(shutdownCtx); err != nil {
			log.Printf("Shutdown hook error: %v", err)
		}
	}

	// Shutdown HTTP server
	if err := s.Server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
		_ = s.Server.Close()
	}

	log.Println("Server stopped")
}

// Shutdown initiates graceful shutdown programmatically.
func (s *GracefulServer) Shutdown(ctx context.Context) error {
	// Execute shutdown hooks
	for i := len(s.shutdownHooks) - 1; i >= 0; i-- {
		hook := s.shutdownHooks[i]
		if err := hook(ctx); err != nil {
			log.Printf("Shutdown hook error: %v", err)
		}
	}

	return s.Server.Shutdown(ctx)
}

// Run starts the server and waits for shutdown signal.
// This is a convenience method that combines Start and WaitForShutdown.
func (s *GracefulServer) Run() error {
	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.Server.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal
	s.WaitForShutdown()

	// Check if server started successfully
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

// RunWithSignalHandler starts the server with a custom signal handler.
func (s *GracefulServer) RunWithSignalHandler(handler func(os.Signal)) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.Server.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for signal or error
	select {
	case err := <-errChan:
		return err
	case sig := <-quit:
		if handler != nil {
			handler(sig)
		}
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	for i := len(s.shutdownHooks) - 1; i >= 0; i-- {
		hook := s.shutdownHooks[i]
		if err := hook(ctx); err != nil {
			log.Printf("Shutdown hook error: %v", err)
		}
	}

	return s.Server.Shutdown(ctx)
}

// ShutdownHooks returns the registered shutdown hooks.
func (s *GracefulServer) ShutdownHooks() []func(context.Context) error {
	return s.shutdownHooks
}

// DefaultShutdownTimeout is the default shutdown timeout.
const DefaultShutdownTimeout = 10 * time.Second

// GracefulShutdown performs a graceful shutdown of the server and its dependencies.
func GracefulShutdown(ctx context.Context, cfg interface{}, engine *gin.Engine, log logger.Logger, postgresDB *gorm.DB, redisClient *database.RedisClient) error {
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	log.Info("Shutting down HTTP server...")
	srv := &http.Server{Handler: engine}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server shutdown error", zap.Error(err))
	}

	// Close database connections
	if postgresDB != nil {
		if sqlDB, err := postgresDB.DB(); err == nil {
			log.Info("Closing database connection...")
			sqlDB.Close()
		}
	}

	if redisClient != nil {
		log.Info("Closing Redis connection...")
		redisClient.Close()
	}

	log.Info("Shutdown complete")
	return nil
}
