package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/linkgen-ai/backend/src/application/services"
	"github.com/linkgen-ai/backend/src/application/usecases"
	"github.com/linkgen-ai/backend/src/application/workers"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/infrastructure/config"
	"github.com/linkgen-ai/backend/src/infrastructure/database"
	dbRepos "github.com/linkgen-ai/backend/src/infrastructure/database/repositories"
	httpServer "github.com/linkgen-ai/backend/src/infrastructure/http"
	"github.com/linkgen-ai/backend/src/infrastructure/http/llm"
	"github.com/linkgen-ai/backend/src/infrastructure/messaging/nats"
	"github.com/linkgen-ai/backend/src/interfaces/handlers"
	"go.uber.org/zap"
)

// Application represents the main application structure
type Application struct {
	// Configuration
	config *config.Config
	logger *zap.Logger

	// Infrastructure
	dbClient   *database.Client
	natsClient *nats.NATSClient
	llmClient  *llm.LLMHTTPClient

	// Repositories
	userRepo  interfaces.UserRepository
	topicRepo interfaces.TopicRepository
	ideaRepo  interfaces.IdeasRepository
	draftRepo interfaces.DraftRepository

	// Use cases
	generateDraftsUC *usecases.GenerateDraftsUseCase
	listIdeasUC      *usecases.ListIdeasUseCase
	clearIdeasUC     *usecases.ClearIdeasUseCase
	refineDraftUC    *usecases.RefineDraftUseCase

	// Workers
	draftWorker *workers.DraftGenerationWorker
	workerCtx   context.Context
	workerCancel context.CancelFunc
	workerWg    sync.WaitGroup

	// HTTP Server
	httpServer *httpServer.Server

	// Worker registry for health checks
	workerRegistry *WorkerRegistry
}

// WorkerRegistry tracks running workers for health monitoring
type WorkerRegistry struct {
	workers map[string]*WorkerStatus
	mu      sync.RWMutex
}

// WorkerStatus represents the status of a worker
type WorkerStatus struct {
	Name      string
	Running   bool
	StartedAt time.Time
	Error     error
}

// NewWorkerRegistry creates a new worker registry
func NewWorkerRegistry() *WorkerRegistry {
	return &WorkerRegistry{
		workers: make(map[string]*WorkerStatus),
	}
}

// Register registers a worker
func (wr *WorkerRegistry) Register(name string) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	wr.workers[name] = &WorkerStatus{
		Name:      name,
		Running:   false,
		StartedAt: time.Time{},
	}
}

// MarkRunning marks a worker as running
func (wr *WorkerRegistry) MarkRunning(name string) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	if status, exists := wr.workers[name]; exists {
		status.Running = true
		status.StartedAt = time.Now()
		status.Error = nil
	}
}

// MarkStopped marks a worker as stopped
func (wr *WorkerRegistry) MarkStopped(name string, err error) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	if status, exists := wr.workers[name]; exists {
		status.Running = false
		status.Error = err
	}
}

// GetStatus returns the status of all workers
func (wr *WorkerRegistry) GetStatus() map[string]*WorkerStatus {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	status := make(map[string]*WorkerStatus)
	for name, ws := range wr.workers {
		status[name] = &WorkerStatus{
			Name:      ws.Name,
			Running:   ws.Running,
			StartedAt: ws.StartedAt,
			Error:     ws.Error,
		}
	}
	return status
}

// IsHealthy returns true if all workers are running
func (wr *WorkerRegistry) IsHealthy() bool {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	
	for _, ws := range wr.workers {
		if !ws.Running {
			return false
		}
	}
	return true
}

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("LinkGen AI - Starting application...")

	// Create application context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize application
	app := &Application{
		logger:         logger,
		workerRegistry: NewWorkerRegistry(),
	}

	if err := app.initialize(ctx); err != nil {
		logger.Fatal("Failed to initialize application", zap.Error(err))
	}

	// Start application
	if err := app.start(ctx); err != nil {
		logger.Fatal("Failed to start application", zap.Error(err))
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.shutdown(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
	}

	logger.Info("Application stopped")
}

// initialize sets up the application dependencies and configuration
func (a *Application) initialize(ctx context.Context) error {
	a.logger.Info("Initializing application dependencies...")

	// Load configuration
	cfg, err := config.LoadFromEnvironment()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	a.config = cfg

	// Initialize database client
	dbConfig := &database.ConnectionConfig{
		URI:                cfg.Database.URI,
		Database:           cfg.Database.Database,
		MinPoolSize:        uint64(cfg.Database.MinPoolSize),
		MaxPoolSize:        uint64(cfg.Database.MaxPoolSize),
		ConnectTimeout:     cfg.Database.ConnectTimeout,
		MaxRetries:         3,
	}
	dbClient := database.GetClient(dbConfig, a.logger)
	a.dbClient = dbClient

	// Connect to database
	if err := dbClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	a.logger.Info("Connected to database")

	// Get collections for repositories
	usersCol, err := dbClient.GetCollection(database.CollectionUsers)
	if err != nil {
		return fmt.Errorf("failed to get users collection: %w", err)
	}
	topicsCol, err := dbClient.GetCollection(database.CollectionTopics)
	if err != nil {
		return fmt.Errorf("failed to get topics collection: %w", err)
	}
	ideasCol, err := dbClient.GetCollection(database.CollectionIdeas)
	if err != nil {
		return fmt.Errorf("failed to get ideas collection: %w", err)
	}
	draftsCol, err := dbClient.GetCollection(database.CollectionDrafts)
	if err != nil {
		return fmt.Errorf("failed to get drafts collection: %w", err)
	}

	// Initialize repositories
	a.userRepo = dbRepos.NewUserRepository(usersCol)
	a.topicRepo = dbRepos.NewTopicRepository(topicsCol)
	a.ideaRepo = dbRepos.NewIdeasRepository(ideasCol)
	a.draftRepo = dbRepos.NewDraftRepository(draftsCol)

	// Initialize LLM client
	llmConfig := llm.Config{
		BaseURL:    cfg.LLM.Endpoint,
		Timeout:    cfg.LLM.Timeout,
		MaxRetries: 3,
		Model:      "gpt-4",
	}
	llmClient, err := llm.NewLLMHTTPClient(llmConfig)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	a.llmClient = llmClient

	// Initialize NATS client
	natsClient, err := nats.NewNATSClient(nats.ClientConfig{
		URL:            cfg.NATS.URL,
		MaxReconnects:  cfg.NATS.MaxReconnects,
		ReconnectDelay: cfg.NATS.ReconnectWait,
		DrainTimeout:   5 * time.Second,
		Logger:         a.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create NATS client: %w", err)
	}
	a.natsClient = natsClient

	// Connect to NATS (non-blocking - workers will retry)
	connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
	defer connectCancel()
	
	if err := natsClient.Connect(connectCtx, 10*time.Second); err != nil {
		a.logger.Warn("Failed to connect to NATS initially, workers will retry", zap.Error(err))
		// Don't fail app startup if NATS is not available
	} else {
		a.logger.Info("Connected to NATS")
	}

	// Initialize use cases
	a.generateDraftsUC = usecases.NewGenerateDraftsUseCase(
		a.userRepo,
		a.ideaRepo,
		a.draftRepo,
		a.llmClient,
	)
	a.listIdeasUC = usecases.NewListIdeasUseCase(a.userRepo, a.ideaRepo)
	a.clearIdeasUC = usecases.NewClearIdeasUseCase(a.userRepo, a.ideaRepo)
	a.refineDraftUC = usecases.NewRefineDraftUseCase(a.draftRepo, a.llmClient)

	// Seed development data
	if err := a.seedDevelopmentData(ctx); err != nil {
		a.logger.Warn("Failed to seed development data", zap.Error(err))
		// Don't fail startup if seeding fails
	}

	// Initialize HTTP server
	if err := a.initializeHTTPServer(); err != nil {
		return fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	// Initialize workers
	if err := a.initializeWorkers(); err != nil {
		return fmt.Errorf("failed to initialize workers: %w", err)
	}

	a.logger.Info("Application dependencies initialized successfully")
	return nil
}

// seedDevelopmentData seeds initial development data
func (a *Application) seedDevelopmentData(ctx context.Context) error {
	a.logger.Info("Seeding development data...")

	seeder := services.NewDevSeeder(a.userRepo, a.topicRepo, a.logger)
	
	if err := seeder.SeedAll(ctx); err != nil {
		return fmt.Errorf("failed to seed development data: %w", err)
	}

	a.logger.Info("Development data seeded successfully")
	return nil
}

// initializeHTTPServer creates and configures the HTTP server with routes
func (a *Application) initializeHTTPServer() error {
	a.logger.Info("Initializing HTTP server...")

	// Create HTTP server
	serverConfig := httpServer.ServerConfig{
		Host:            a.config.Server.Host,
		Port:            a.config.Server.Port,
		ReadTimeout:     a.config.Server.ReadTimeout,
		WriteTimeout:    a.config.Server.WriteTimeout,
		ShutdownTimeout: a.config.Server.ShutdownTimeout,
	}
	a.httpServer = httpServer.NewServer(serverConfig, a.logger)

	// Get router
	router := a.httpServer.GetRouter()

	// Create NATS publisher for drafts
	draftPublisher, err := nats.NewPublisher(nats.PublisherConfig{
		Client:  a.natsClient,
		Subject: "draft.generate",
		Logger:  a.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create NATS publisher: %w", err)
	}

	// Create adapter for database health checker
	dbHealthChecker := &dbHealthAdapter{client: a.dbClient}
	
	// Create adapter for worker registry
	workerRegistryAdapter := &workerRegistryAdapter{registry: a.workerRegistry}
	
	// Register health handler
	healthHandler := handlers.NewHealthHandler(
		dbHealthChecker,
		workerRegistryAdapter,
		a.natsClient,
		a.logger,
	)
	router.HandleFunc("/health", healthHandler.HandleHealth).Methods("GET")
	router.HandleFunc("/readiness", healthHandler.HandleReadiness).Methods("GET")
	router.HandleFunc("/liveness", healthHandler.HandleLiveness).Methods("GET")

	// Register topics handler
	topicsHandler := handlers.NewTopicsHandler(
		a.topicRepo,
		a.userRepo,
		a.logger,
	)
	topicsHandler.RegisterRoutes(router)

	// Register ideas handler
	ideasHandler := handlers.NewIdeasHandler(
		a.listIdeasUC,
		a.clearIdeasUC,
		a.logger,
	)
	ideasHandler.RegisterRoutes(router)

	// Register drafts handler
	draftsHandler := handlers.NewDraftsHandler(
		a.refineDraftUC,
		a.draftRepo,
		draftPublisher,
		a.logger,
	)
	draftsHandler.RegisterRoutes(router)

	a.logger.Info("HTTP server initialized successfully")
	return nil
}

// initializeWorkers creates and configures workers
func (a *Application) initializeWorkers() error {
	a.logger.Info("Initializing workers...")

	// Create NATS consumer for draft generation
	consumer, err := nats.NewConsumer(nats.ConsumerConfig{
		Client:        a.natsClient,
		Subject:       "draft.generate",
		QueueGroup:    "draft-workers",
		MaxConcurrent: 1,
		MaxRetries:    2,
		Logger:        a.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create NATS consumer: %w", err)
	}

	// Create use case adapter
	ucAdapter := &useCaseAdapter{
		useCase: a.generateDraftsUC,
	}

	// Create draft generation worker
	draftWorker, err := workers.NewDraftGenerationWorker(workers.WorkerConfig{
		Consumer:   consumer,
		UseCase:    ucAdapter,
		MaxRetries: 2,
		Logger:     a.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create draft generation worker: %w", err)
	}
	a.draftWorker = draftWorker

	// Register worker in registry
	a.workerRegistry.Register("draft_generation")

	a.logger.Info("Workers initialized successfully")
	return nil
}

// start begins the application services (HTTP server, scheduler, workers)
func (a *Application) start(ctx context.Context) error {
	a.logger.Info("Starting application services...")

	// Start HTTP server
	if err := a.httpServer.Start(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	// Create worker context
	a.workerCtx, a.workerCancel = context.WithCancel(ctx)

	// Start workers as goroutines
	if err := a.startWorkers(a.workerCtx); err != nil {
		return fmt.Errorf("failed to start workers: %w", err)
	}

	// TODO: Start scheduler
	
	a.logger.Info("Application services started successfully")
	fmt.Printf("LinkGen AI is running on http://%s:%d\n", a.config.Server.Host, a.config.Server.Port)
	
	return nil
}

// startWorkers starts all workers as goroutines
func (a *Application) startWorkers(ctx context.Context) error {
	a.logger.Info("Starting workers...")

	// Start draft generation worker
	a.workerWg.Add(1)
	go func() {
		defer a.workerWg.Done()
		
		a.logger.Info("Starting draft generation worker...")
		
		if err := a.draftWorker.Start(ctx); err != nil {
			a.logger.Error("Draft generation worker failed to start", zap.Error(err))
			a.workerRegistry.MarkStopped("draft_generation", err)
			return
		}
		
		a.workerRegistry.MarkRunning("draft_generation")
		a.logger.Info("Draft generation worker started successfully")
		
		// Wait for context cancellation
		<-ctx.Done()
		a.logger.Info("Draft generation worker context cancelled")
	}()

	// Give workers a moment to start
	time.Sleep(100 * time.Millisecond)

	a.logger.Info("Workers started successfully")
	return nil
}

// stopWorkers stops all workers gracefully
func (a *Application) stopWorkers(timeout time.Duration) error {
	a.logger.Info("Stopping workers...", zap.Duration("timeout", timeout))

	// Cancel worker context
	if a.workerCancel != nil {
		a.workerCancel()
	}

	// Stop draft worker
	if a.draftWorker != nil {
		if err := a.draftWorker.Stop(timeout); err != nil {
			a.logger.Warn("Failed to stop draft generation worker cleanly", zap.Error(err))
			a.workerRegistry.MarkStopped("draft_generation", err)
		} else {
			a.workerRegistry.MarkStopped("draft_generation", nil)
		}
	}

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		a.workerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Info("All workers stopped successfully")
		return nil
	case <-time.After(timeout):
		a.logger.Warn("Worker shutdown timeout exceeded")
		return fmt.Errorf("worker shutdown timeout exceeded")
	}
}

// shutdown performs graceful shutdown of all services
func (a *Application) shutdown(ctx context.Context) error {
	a.logger.Info("Shutting down application services...")

	// Stop HTTP server first
	if a.httpServer != nil {
		if err := a.httpServer.Shutdown(ctx); err != nil {
			a.logger.Error("Error stopping HTTP server", zap.Error(err))
		}
	}

	// Stop workers
	workerShutdownTimeout := 5 * time.Second
	if err := a.stopWorkers(workerShutdownTimeout); err != nil {
		a.logger.Error("Error stopping workers", zap.Error(err))
	}

	// TODO: Stop scheduler

	// Disconnect from NATS
	if a.natsClient != nil {
		if err := a.natsClient.Disconnect(5 * time.Second); err != nil {
			a.logger.Warn("Failed to disconnect from NATS cleanly", zap.Error(err))
		}
	}

	// Disconnect from database
	if a.dbClient != nil {
		if err := a.dbClient.Disconnect(ctx); err != nil {
			a.logger.Warn("Failed to disconnect from database cleanly", zap.Error(err))
		}
	}

	a.logger.Info("Application shutdown complete")
	return nil
}

// GetWorkerRegistry returns the worker registry for health checks
func (a *Application) GetWorkerRegistry() *WorkerRegistry {
	return a.workerRegistry
}

// useCaseAdapter adapts usecases.GenerateDraftsUseCase to workers.GenerateDraftsUseCase
type useCaseAdapter struct {
	useCase *usecases.GenerateDraftsUseCase
}

// Execute adapts the interface
func (uca *useCaseAdapter) Execute(ctx context.Context, input workers.GenerateDraftsInput) ([]*workers.Draft, error) {
	// Convert input
	ucInput := usecases.GenerateDraftsInput{
		UserID: input.UserID,
		IdeaID: input.IdeaID,
	}

	// Execute use case
	drafts, err := uca.useCase.Execute(ctx, ucInput)
	if err != nil {
		return nil, err
	}

	// Convert output
	workerDrafts := make([]*workers.Draft, len(drafts))
	for i, d := range drafts {
		workerDrafts[i] = &workers.Draft{
			ID: d.ID,
		}
	}

	return workerDrafts, nil
}

// dbHealthAdapter adapts database.Client to handlers.HealthChecker
type dbHealthAdapter struct {
	client *database.Client
}

// Check adapts the health check interface
func (a *dbHealthAdapter) Check(ctx context.Context) *handlers.HealthCheckResult {
	result := a.client.Check(ctx)
	return &handlers.HealthCheckResult{
		Status:  string(result.Status),
		Latency: result.Latency,
		Error:   result.Error,
	}
}

// workerRegistryAdapter adapts WorkerRegistry to handlers.WorkerRegistry
type workerRegistryAdapter struct {
	registry *WorkerRegistry
}

// GetStatus adapts the worker status interface
func (a *workerRegistryAdapter) GetStatus() map[string]*handlers.WorkerStatus {
	status := a.registry.GetStatus()
	adapted := make(map[string]*handlers.WorkerStatus)
	for name, ws := range status {
		adapted[name] = &handlers.WorkerStatus{
			Name:      ws.Name,
			Running:   ws.Running,
			StartedAt: ws.StartedAt,
			Error:     ws.Error,
		}
	}
	return adapted
}

// IsHealthy adapts the health check interface
func (a *workerRegistryAdapter) IsHealthy() bool {
	return a.registry.IsHealthy()
}
