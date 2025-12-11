package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	domainErrors "github.com/linkgen-ai/backend/src/domain/errors"
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

// JobErrorRepository defines how job errors are persisted
type JobErrorRepository interface {
	Create(ctx context.Context, jobError *JobError) (string, error)
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

// JobError captures failure diagnostics for a job execution
type JobError struct {
	JobID       string
	UserID      string
	IdeaID      string
	Stage       string
	Error       string
	RawResponse string
	Prompt      string
	Attempt     int
}

const jobErrorStageDraftGeneration = "draft_generation"

// DraftGenerationWorker handles async draft generation from NATS queue
type DraftGenerationWorker struct {
	consumer     *nats.Consumer
	useCase      GenerateDraftsUseCase
	jobRepo      JobRepository
	jobErrorRepo JobErrorRepository
	logger       *zap.Logger
	mu           sync.RWMutex
	running      bool
	maxRetries   int
	// Metrics
	messagesProcessedTotal  int64
	processingErrorsTotal   int64
	retriesTotal            int64
	generationFailuresTotal int64
}

// WorkerConfig holds worker configuration
type WorkerConfig struct {
	Consumer     *nats.Consumer
	UseCase      GenerateDraftsUseCase
	JobRepo      JobRepository
	JobErrorRepo JobErrorRepository
	MaxRetries   int
	Logger       *zap.Logger
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
		consumer:     config.Consumer,
		useCase:      config.UseCase,
		jobRepo:      config.JobRepo,
		jobErrorRepo: config.JobErrorRepo,
		logger:       logger,
		maxRetries:   maxRetries,
		running:      false,
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
		w.incrementProcessingErrors()
		w.logger.Error("failed to parse message", zap.Error(err))
		return nil
	}

	// Validate message
	if err := w.validateMessage(&msg); err != nil {
		w.incrementProcessingErrors()
		w.logger.Error("invalid draft generation message",
			zap.String("job_id", msg.JobID),
			zap.Error(err),
		)
		return nil
	}

	defer w.incrementMessagesProcessed()

	w.logger.Info("processing draft generation request",
		zap.String("user_id", msg.UserID),
		zap.String("idea_id", msg.IdeaID),
		zap.String("job_id", msg.JobID),
	)

	w.markJobProcessing(ctx, msg.JobID)

	drafts, err := w.generateWithRetries(ctx, msg)
	if err != nil {
		w.logger.Error("draft generation failed after retries",
			zap.String("user_id", msg.UserID),
			zap.String("idea_id", msg.IdeaID),
			zap.String("job_id", msg.JobID),
			zap.Error(err),
		)
		w.markJobFailed(ctx, msg.JobID, err)
		return nil
	}

	w.markJobCompleted(ctx, msg.JobID, drafts)

	w.logger.Info("draft generation completed successfully",
		zap.String("user_id", msg.UserID),
		zap.String("idea_id", msg.IdeaID),
		zap.String("job_id", msg.JobID),
		zap.Int("drafts_generated", len(drafts)),
	)

	return nil
}

// validateMessage ensures that the incoming message contains the required fields
func (w *DraftGenerationWorker) validateMessage(msg *DraftGenerationMessage) error {
	if msg.UserID == "" {
		return errors.New("user_id is required")
	}

	if msg.IdeaID == "" {
		return errors.New("idea_id is required")
	}

	if msg.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}

	return nil
}

// generateWithRetries executes the draft generation use case with in-process retries
func (w *DraftGenerationWorker) generateWithRetries(ctx context.Context, msg DraftGenerationMessage) ([]*Draft, error) {
	totalAttempts := w.maxRetries + 1
	var lastErr error

	for attempt := 0; attempt < totalAttempts; attempt++ {
		if ctx.Err() != nil {
			lastErr = ctx.Err()
			break
		}

		drafts, err := w.useCase.Execute(ctx, GenerateDraftsInput{
			UserID: msg.UserID,
			IdeaID: msg.IdeaID,
		})
		if err == nil {
			return drafts, nil
		}

		lastErr = err
		w.incrementProcessingErrors()

		if attempt < w.maxRetries {
			w.incrementRetries()
			w.logger.Warn("draft generation attempt failed, retrying",
				zap.String("job_id", msg.JobID),
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", totalAttempts),
				zap.Error(err),
			)
			continue
		}
	}

	w.incrementGenerationFailures()
	var llmRespErr *domainErrors.LLMResponseError
	if errors.As(lastErr, &llmRespErr) {
		w.recordJobError(ctx, msg, totalAttempts, llmRespErr)
	}
	return nil, fmt.Errorf("draft generation failed after %d attempts: %w", totalAttempts, lastErr)
}

// recordJobError persists detailed error information for troubleshooting
func (w *DraftGenerationWorker) recordJobError(ctx context.Context, msg DraftGenerationMessage, attempt int, llmErr *domainErrors.LLMResponseError) {
	if w.jobErrorRepo == nil || llmErr == nil {
		return
	}

	ideaID := msg.IdeaID
	jobErr := &JobError{
		JobID:       msg.JobID,
		UserID:      msg.UserID,
		IdeaID:      ideaID,
		Stage:       jobErrorStageDraftGeneration,
		Error:       llmErr.Reason,
		RawResponse: llmErr.RawResponse,
		Prompt:      llmErr.Prompt,
		Attempt:     attempt,
	}

	if jobErr.Error == "" && llmErr.Err != nil {
		jobErr.Error = llmErr.Err.Error()
	}
	if jobErr.Error == "" {
		jobErr.Error = "unknown llm response error"
	}

	if _, err := w.jobErrorRepo.Create(ctx, jobErr); err != nil {
		w.logger.Warn("failed to persist job error",
			zap.String("job_id", msg.JobID),
			zap.Error(err),
		)
	}
}

// markJobProcessing transitions a job to processing status if possible
func (w *DraftGenerationWorker) markJobProcessing(ctx context.Context, jobID string) {
	w.updateJob(ctx, jobID, func(job *Job) bool {
		if job.Status == "processing" {
			return false
		}

		now := time.Now()
		job.Status = "processing"
		job.StartedAt = &now
		job.UpdatedAt = now
		return true
	})
}

// markJobCompleted updates the job with completion metadata and generated draft IDs
func (w *DraftGenerationWorker) markJobCompleted(ctx context.Context, jobID string, drafts []*Draft) {
	if len(drafts) == 0 {
		w.logger.Warn("no drafts generated but job marked as completed",
			zap.String("job_id", jobID),
		)
	}

	draftIDs := make([]string, len(drafts))
	for i, draft := range drafts {
		draftIDs[i] = draft.ID
	}

	w.updateJob(ctx, jobID, func(job *Job) bool {
		now := time.Now()
		job.Status = "completed"
		job.CompletedAt = &now
		job.UpdatedAt = now
		job.DraftIDs = draftIDs
		job.Error = ""
		return true
	})
}

// markJobFailed records the failure details for the job
func (w *DraftGenerationWorker) markJobFailed(ctx context.Context, jobID string, failure error) {
	if failure == nil {
		return
	}

	message := failure.Error()
	if len(message) > 2048 {
		message = message[:2048]
	}

	w.updateJob(ctx, jobID, func(job *Job) bool {
		now := time.Now()
		job.Status = "failed"
		job.Error = message
		job.CompletedAt = &now
		job.UpdatedAt = now
		return true
	})
}

// updateJob loads a job, applies the provided mutation, and persists the change when needed
func (w *DraftGenerationWorker) updateJob(ctx context.Context, jobID string, mutate func(job *Job) bool) {
	if w.jobRepo == nil {
		return
	}

	job, err := w.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		w.logger.Warn("failed to load job for update",
			zap.String("job_id", jobID),
			zap.Error(err),
		)
		return
	}

	if job == nil {
		w.logger.Warn("job not found for update",
			zap.String("job_id", jobID),
		)
		return
	}

	if !mutate(job) {
		return
	}

	if err := w.jobRepo.Update(ctx, job); err != nil {
		w.logger.Warn("failed to update job",
			zap.String("job_id", jobID),
			zap.Error(err),
		)
	}
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
