package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// Application represents the main application structure
type Application struct {
	// DI container and dependencies will be injected here
}

func main() {
	// Initialize logger
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	log.Info("LinkGen AI - Starting application...")

	// Create application context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize application
	app := &Application{}
	if err := app.initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Start application
	if err := app.start(ctx); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down gracefully...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.shutdown(shutdownCtx); err != nil {
		log.Errorf("Error during shutdown: %v", err)
	}

	log.Info("Application stopped")
}

// initialize sets up the application dependencies and configuration
func (a *Application) initialize(ctx context.Context) error {
	log.Info("Initializing application dependencies...")
	// TODO: Initialize configuration, database, NATS, LLM client, etc.
	return nil
}

// start begins the application services (HTTP server, scheduler, workers)
func (a *Application) start(ctx context.Context) error {
	log.Info("Starting application services...")
	// TODO: Start HTTP server, scheduler, NATS workers
	fmt.Println("LinkGen AI is running on http://localhost:8080")
	return nil
}

// shutdown performs graceful shutdown of all services
func (a *Application) shutdown(ctx context.Context) error {
	log.Info("Shutting down application services...")
	// TODO: Stop HTTP server, scheduler, close DB connections, close NATS
	return nil
}
