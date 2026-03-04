// Package server provides HTTP server utilities for Gin-based microservices.
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Config holds server configuration.
type Config struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// Server wraps http.Server with additional functionality.
type Server struct {
	*http.Server
	engine          *gin.Engine
	shutdownTimeout time.Duration
	onShutdown      []func(context.Context) error
}

// New creates a new HTTP server with Gin engine.
func New(cfg Config) *Server {
	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      engine,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return &Server{
		Server:          srv,
		engine:          engine,
		shutdownTimeout: cfg.ShutdownTimeout,
		onShutdown:      make([]func(context.Context) error, 0),
	}
}

// Engine returns the underlying Gin engine.
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// OnShutdown registers a function to be called during shutdown.
func (s *Server) OnShutdown(fn func(context.Context) error) {
	s.onShutdown = append(s.onShutdown, fn)
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

// StartTLS starts the HTTP server with TLS.
func (s *Server) StartTLS(certFile, keyFile string) error {
	if err := s.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start TLS server: %w", err)
	}
	return nil
}

// Router returns the Gin engine for route registration.
func (s *Server) Router() *gin.Engine {
	return s.engine
}

// Use adds middleware to the Gin engine.
func (s *Server) Use(middleware ...gin.HandlerFunc) {
	s.engine.Use(middleware...)
}

// Group creates a router group.
func (s *Server) Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return s.engine.Group(relativePath, handlers...)
}

// Any registers a route that matches all HTTP methods.
func (s *Server) Any(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.Any(relativePath, handlers...)
}

// GET registers a GET route.
func (s *Server) GET(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.GET(relativePath, handlers...)
}

// POST registers a POST route.
func (s *Server) POST(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.POST(relativePath, handlers...)
}

// PUT registers a PUT route.
func (s *Server) PUT(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.PUT(relativePath, handlers...)
}

// PATCH registers a PATCH route.
func (s *Server) PATCH(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.PATCH(relativePath, handlers...)
}

// DELETE registers a DELETE route.
func (s *Server) DELETE(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.DELETE(relativePath, handlers...)
}

// Static registers a static file route.
func (s *Server) Static(relativePath, root string) {
	s.engine.Static(relativePath, root)
}

// StaticFile registers a single static file route.
func (s *Server) StaticFile(relativePath, filepath string) {
	s.engine.StaticFile(relativePath, filepath)
}

// LoadHTMLGlob loads HTML templates.
func (s *Server) LoadHTMLGlob(pattern string) {
	s.engine.LoadHTMLGlob(pattern)
}

// NoRoute sets handlers for no route.
func (s *Server) NoRoute(handlers ...gin.HandlerFunc) {
	s.engine.NoRoute(handlers...)
}

// NoMethod sets handlers for no method.
func (s *Server) NoMethod(handlers ...gin.HandlerFunc) {
	s.engine.NoMethod(handlers...)
}

// NewGinEngine creates a new Gin engine with default middleware.
func NewGinEngine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	return engine
}
