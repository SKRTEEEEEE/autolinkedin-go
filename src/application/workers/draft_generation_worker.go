package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/linkgen-ai/backend/src/infrastructure/messaging/nats"
	"go.uber.org/zap"
)

var (
	// ErrNilUseCase indicates nil use case
	ErrNilUseCase = errors.New("use case cannot be nil")
	// ErrNilConsumer indicates nil consumer
	ErrNilConsumer = errors.New("consumer cannot be nil")
	// ErrAlreadyRunning indicates worker is already running
	ErrAlreadyRunning = errors.New("worker already running")
	// ErrNotRunning indicates worker is not running
	ErrNotRunning = errors.New("worker not running")
)

// DraftGenerationMessage represents the message structure for draft generation
type DraftGenerationMessage struct {
	JobID      string    `json:"job_id"`
	UserID     string    `json:"user_id"`
	IdeaID     string    `json:"idea_id"`
	Timestamp  time.Time `json:"timestamp"`
	RetryCount int       `json:"retry_count"`
}

// GenerateDraftsUseCase is the interface for the draft generation use case
type GenerateDraftsUseCase interface {
	Execute(ctx context.Context, input GenerateDraftsInput) ([]*Draft, error)
}

// GenerateDraftsInput represents input for draft generation
type GenerateDraftsInput struct {
	UserID string
	IdeaID string
}

// Draft is a minimal draft entity representation for workers
type Draft struct {
	ID string
}

// JobRepository defines the interface for job persistence
type JobRepository interface {
	FindByID(ctx context.Context, jobID string) (*Job, error)
	Update(ctx context.Context, job *Job) error
}

// Job is a minimal job entity representation for workers
type Job struct {
	ID          string
	UserID      string
	Type        string
	Status      string
	IdeaID      *string
	DraftIDs    []string
	Error       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// DraftGenerationWorker handles async draft generation from NATS queue
type DraftGenerationWorker struct {
	consumer    *nats.Consumer
	useCase     GenerateDraftsUseCase
	jobRepo     JobRepository
	logger      *zap.Logger
	mu          sync.RWMutex
	running     bool
	maxRetries  int
	// Metrics
	messagesProcessedTotal  int64
	processingErrorsTotal   int64
	retriesTotal            int64
	generationFailuresTotal int64
}

// WorkerConfig holds worker configuration
type WorkerConfig struct {
	Consumer   *nats.Consumer
	UseCase    GenerateDraftsUseCase
	JobRepo    JobRepository
	MaxRetries int
	Logger     *zap.Logger
}

// NewDraftGenerationWorker creates a new draft generation worker
func NewDraftGenerationWorker(config WorkerConfig) (*DraftGenerationWorker, error) {
	if config.UseCase == nil {
		return nil, ErrNilUseCase
	}

	if config.Consumer == nil {
		return nil, ErrNilConsumer
	}

	logger := config.Logger
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2 // Default from requirements
	}

	return &DraftGenerationWorker{
		consumer:   config.Consumer,
		useCase:    config.UseCase,
		jobRepo:    config.JobRepo,
		logger:     logger,
		maxRetries: maxRetries,
		running:    false,
	}, nil
}

// Start starts the worker
func (w *DraftGenerationWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return ErrAlreadyRunning
	}

	// Subscribe to consumer with message handler
	err := w.consumer.Subscribe(ctx, w.processMessage)
	if err != nil {
		return fmt.Errorf("failed to start worker: %w", err)
	}

	w.running = true
	w.logger.Info("draft generation worker started")

	return nil
}

// Stop stops the worker gracefully
func (w *DraftGenerationWorker) Stop(shutdownTimeout time.Duration) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return ErrNotRunning
	}

	// Unsubscribe from consumer
	err := w.consumer.Unsubscribe(shutdownTimeout)
	if err != nil {
		w.logger.Warn("failed to unsubscribe consumer", zap.Error(err))
		return err
	}

	w.running = false
	w.logger.Info("draft generation worker stopped")

	return nil
}

// processMessage processes a single draft generation message
func (w *DraftGenerationWorker) processMessage(ctx context.Context, msgData []byte) error {
	// Parse message
	var msg DraftGenerationMessage
	if err := json.Unmarshal(msgData, &msg); err != nil {
		w.logger.Error("failed to parse message", zap.Error(err))
		return fmt.Errorf("invalid message format: %w", err)
	}

	// Validate message
	if msg.UserID == "" {
		w.incrementProcessingErrors()
		return errors.New("user_id is required")
	}

	if msg.IdeaID == "" {
		w.incrementProcessingErrors()
		return errors.New("idea_id is required")
	}

	if msg.Timestamp.IsZero() {
		w.incrementProcessingErrors()
		return errors.New("timestamp is required")
	}

	w.logger.Info("processing draft generation request",
		zap.String("user_id", msg.UserID),
		zap.String("idea_id", msg.IdeaID),
		zap.String("job_id", msg.JobID),
		zap.Int("retry_count", msg.RetryCount),
	)

	// Update job status to processing if jobRepo is available
	if w.jobRepo != nil {
		job, err := w.jobRepo.FindByID(ctx, msg.JobID)
		if err == nil && job != nil && job.Status == "pending" {
			now := time.Now()
			job.Status = "processing"
			job.StartedAt = &now
			job.UpdatedAt = now
			_ = w.jobRepo.Update(ctx, job)
		}
	}

	// Execute use case
	drafts, err := w.useCase.Execute(ctx, GenerateDraftsInput{
		UserID: msg.UserID,
		IdeaID: msg.IdeaID,
	})
	if err != nil {
		w.incrementProcessingErrors()
		w.logger.Error("use case execution failed",
			zap.String("user_id", msg.UserID),
			zap.String("idea_id", msg.IdeaID),
			zap.String("job_id", msg.JobID),
			zap.Error(err),
		)

		// Check if we should retry
		if msg.RetryCount < w.maxRetries {
			w.incrementRetries()
			return fmt.Errorf("draft generation failed (retry %d/%d): %w",
				msg.RetryCount, w.maxRetries, err)
		}

		// Max retries reached - mark as failed
		w.incrementGenerationFailures()
		w.logger.Error("max retries reached, marking job as failed",
			zap.String("user_id", msg.UserID),
			zap.String("idea_id", msg.IdeaID),
			zap.String("job_id", msg.JobID),
			zap.Int("retry_count", msg.RetryCount),
		)

		// Update job status to failed
		if w.jobRepo != nil {
			job, findErr := w.jobRepo.FindByID(ctx, msg.JobID)
			if findErr == nil && job != nil {
				now := time.Now()
				job.Status = "failed"
				job.Error = err.Error()
				job.CompletedAt = &now
				job.UpdatedAt = now
				_ = w.jobRepo.Update(ctx, job)
			}
		}

		return fmt.Errorf("draft generation failed after %d retries: %w",
			msg.RetryCount, err)
	}

	// Update job status to completed
	if w.jobRepo != nil {
		job, findErr := w.jobRepo.FindByID(ctx, msg.JobID)
		if findErr == nil && job != nil {
			now := time.Now()
			job.Status = "completed"
			job.CompletedAt = &now
			job.UpdatedAt = now
			
			// Extract draft IDs
			if len(drafts) > 0 {
				job.DraftIDs = make([]string, len(drafts))
				for i, draft := range drafts {
					job.DraftIDs[i] = draft.ID
				}
			}
			
			_ = w.jobRepo.Update(ctx, job)
		}
	}

	w.incrementMessagesProcessed()
	w.logger.Info("draft generation completed successfully",
		zap.String("user_id", msg.UserID),
		zap.String("idea_id", msg.IdeaID),
		zap.String("job_id", msg.JobID),
		zap.Int("drafts_generated", len(drafts)),
	)

	return nil
}

// IsRunning returns worker running status
func (w *DraftGenerationWorker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetMetrics returns worker metrics
func (w *DraftGenerationWorker) GetMetrics() map[string]int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return map[string]int64{
		"messages_processed_total":  w.messagesProcessedTotal,
		"processing_errors_total":   w.processingErrorsTotal,
		"retries_total":             w.retriesTotal,
		"generation_failures_total": w.generationFailuresTotal,
	}
}

// incrementMessagesProcessed increments processed messages counter
func (w *DraftGenerationWorker) incrementMessagesProcessed() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messagesProcessedTotal++
}

// incrementProcessingErrors increments processing errors counter
func (w *DraftGenerationWorker) incrementProcessingErrors() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.processingErrorsTotal++
}

// incrementRetries increments retries counter
func (w *DraftGenerationWorker) incrementRetries() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.retriesTotal++
}

// incrementGenerationFailures increments generation failures counter
func (w *DraftGenerationWorker) incrementGenerationFailures() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.generationFailuresTotal++
}
