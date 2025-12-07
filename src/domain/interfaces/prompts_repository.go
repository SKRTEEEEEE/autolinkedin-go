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
}
