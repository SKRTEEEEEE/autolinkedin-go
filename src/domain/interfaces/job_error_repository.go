package interfaces

import (
	"context"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// JobErrorRepository defines persistence operations for job errors
type JobErrorRepository interface {
	Create(ctx context.Context, jobError *entities.JobError) (string, error)
}
