package interfaces

import (
	"context"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// DraftRepository defines the interface for draft persistence operations
type DraftRepository interface {
	// Create creates a new draft
	Create(ctx context.Context, draft *entities.Draft) (string, error)

	// FindByID retrieves a draft by its unique ID
	FindByID(ctx context.Context, draftID string) (*entities.Draft, error)

	// Update updates draft information
	Update(ctx context.Context, draftID string, updates map[string]interface{}) error

	// Delete removes a draft from the system
	Delete(ctx context.Context, draftID string) error

	// ListByUserID retrieves drafts for a user with optional filtering
	// status: filter by draft status (empty string for all statuses)
	// draftType: filter by type POST/ARTICLE (empty string for all types)
	ListByUserID(ctx context.Context, userID string, status entities.DraftStatus, draftType entities.DraftType) ([]*entities.Draft, error)

	// UpdateStatus updates the status of a draft
	UpdateStatus(ctx context.Context, draftID string, status entities.DraftStatus) error

	// AppendRefinement adds a refinement entry to a draft
	AppendRefinement(ctx context.Context, draftID string, entry entities.RefinementEntry) error

	// FindReadyForPublishing retrieves drafts ready to be published
	FindReadyForPublishing(ctx context.Context, userID string) ([]*entities.Draft, error)
}
