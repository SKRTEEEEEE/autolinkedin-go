package interfaces

import (
	"context"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// JobRepository defines the interface for job persistence operations
type JobRepository interface {
	// Create creates a new job and returns its ID
	Create(ctx context.Context, job *entities.Job) (string, error)

	// FindByID retrieves a job by its unique ID
	FindByID(ctx context.Context, jobID string) (*entities.Job, error)

	// Update updates an existing job
	Update(ctx context.Context, job *entities.Job) error

	// ListByUserID retrieves all jobs for a specific user
	// Returns jobs ordered by creation date (most recent first)
	ListByUserID(ctx context.Context, userID string, limit int) ([]*entities.Job, error)
}
