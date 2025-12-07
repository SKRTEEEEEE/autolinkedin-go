package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Server represents an HTTP server
type Server struct {
	router     *mux.Router
	httpServer *http.Server
	logger     *zap.Logger
	config     ServerConfig
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// NewServer creates a new HTTP server
func NewServer(config ServerConfig, logger *zap.Logger) *Server {
	router := mux.NewRouter()

	return &Server{
		router: router,
		logger: logger,
		config: config,
	}
}

// GetRouter returns the mux router for registering routes
func (s *Server) GetRouter() *mux.Router {
	return s.router
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	s.logger.Info("Starting HTTP server", zap.String("address", addr))

	// Start server in goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	s.logger.Info("Shutting down HTTP server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP server shutdown failed: %w", err)
	}

	s.logger.Info("HTTP server stopped successfully")
	return nil
}
