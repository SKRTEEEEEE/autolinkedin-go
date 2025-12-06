package interfaces

import (
	"context"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// TopicRepository defines the interface for topic persistence operations
type TopicRepository interface {
	// Create creates a new topic for a user
	Create(ctx context.Context, topic *entities.Topic) (string, error)

	// FindByID retrieves a topic by its unique ID
	FindByID(ctx context.Context, topicID string) (*entities.Topic, error)

	// ListByUserID retrieves all topics belonging to a specific user
	ListByUserID(ctx context.Context, userID string) ([]*entities.Topic, error)

	// FindRandomByUserID selects a random topic from user's topics
	// Used by the scheduler for periodic idea generation
	FindRandomByUserID(ctx context.Context, userID string) (*entities.Topic, error)

	// Delete removes a topic from the system
	Delete(ctx context.Context, topicID string) error
}
