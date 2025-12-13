package interfaces

import (
	"context"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// PromptsRepository defines methods for prompt persistence
type PromptsRepository interface {
	// Create creates a new prompt and returns its ID
	Create(ctx context.Context, prompt *entities.Prompt) (string, error)

	// FindByID retrieves a prompt by its ID
	FindByID(ctx context.Context, id string) (*entities.Prompt, error)

	// FindByName retrieves a prompt by its name for a specific user
	FindByName(ctx context.Context, userID string, name string) (*entities.Prompt, error)

	// ListByUserID retrieves all prompts for a user
	ListByUserID(ctx context.Context, userID string) ([]*entities.Prompt, error)

	// ListByUserIDAndType retrieves prompts for a user filtered by type
	ListByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error)

	// FindActiveByUserIDAndType retrieves active prompts for a user filtered by type
	FindActiveByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error)

	// FindByUserIDAndStyle retrieves a prompt by user and style name (for drafts)
	FindByUserIDAndStyle(ctx context.Context, userID string, styleName string) (*entities.Prompt, error)

	// Update updates an existing prompt
	Update(ctx context.Context, prompt *entities.Prompt) error

	// Delete removes a prompt by ID
	Delete(ctx context.Context, id string) error

	// CountByUserID returns the count of prompts for a user
	CountByUserID(ctx context.Context, userID string) (int64, error)

	// --- Additional methods for prompt synchronization and management ---

	// CreateBatch creates multiple prompts in a single operation
	CreateBatch(ctx context.Context, prompts []*entities.Prompt) ([]string, error)

	// FindOrCreateByName retrieves a prompt by name for a user, creates it if not found
	FindOrCreateByName(ctx context.Context, userID string, name string, promptType entities.PromptType, template string) (*entities.Prompt, error)

	// ListActiveByUserIDAndType retrieves active prompts for a user filtered by type (alias for FindActiveByUserIDAndType)
	ListActiveByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error)

	// DeactivateByUserIDAndName deactivates a prompt by user and name
	DeactivateByUserIDAndName(ctx context.Context, userID string, name string) error

	// Upsert updates a prompt or creates it if it doesn't exist
	Upsert(ctx context.Context, prompt *entities.Prompt) (string, error)
}
