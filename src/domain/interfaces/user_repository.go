package interfaces

import (
	"context"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// UserRepository defines the interface for user persistence operations
type UserRepository interface {
	// Create creates a new user in the system
	Create(ctx context.Context, user *entities.User) (string, error)

	// FindByID retrieves a user by their unique ID
	FindByID(ctx context.Context, userID string) (*entities.User, error)

	// FindByEmail retrieves a user by their email address
	FindByEmail(ctx context.Context, email string) (*entities.User, error)

	// Update updates user information
	Update(ctx context.Context, userID string, updates map[string]interface{}) error

	// UpdateLinkedInToken updates the LinkedIn access token for a user
	UpdateLinkedInToken(ctx context.Context, userID string, token string) error

	// Delete removes a user from the system
	Delete(ctx context.Context, userID string) error
}
