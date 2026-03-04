// Package main is the entry point for the user service.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ignata/go-microservices-boilerplate/internal/user/delivery"
	"github.com/ignata/go-microservices-boilerplate/pkg/config"
	pkgserver "github.com/ignata/go-microservices-boilerplate/pkg/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Initialize application with Wire
	app, err := initializeApp(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize application: %v", err))
	}

	// Register routes
	handler := delivery.NewUserHandler(app.UserUseCase)
	delivery.RegisterRoutes(app.Engine, handler)

	// Add health check endpoints
	pkgserver.RegisterHealthRoutes(app.Engine, app.PostgresDB.DB, app.RedisClient)

	// Add recovery middleware
	app.Engine.Use(gin.Recovery())

	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		app.Log.Info(fmt.Sprintf("Starting user service on %s", addr))

		srv := &http.Server{
			Addr:         addr,
			Handler:      app.Engine,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Log.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		app.Log.Error("Server forced to shutdown", zap.Error(err))
	}

	app.Log.Info("Server exited")
}
