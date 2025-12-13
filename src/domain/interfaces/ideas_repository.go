package interfaces

import (
	"context"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// IdeasRepository defines the interface for ideas persistence operations
type IdeasRepository interface {
	// CreateBatch creates multiple ideas at once
	// Used by the scheduler when generating periodic ideas
	CreateBatch(ctx context.Context, ideas []*entities.Idea) error

	// ListByUserID retrieves ideas for a user with optional filtering
	// topicID: filter by specific topic (empty string for all topics)
	// limit: maximum number of ideas to return (0 for no limit)
	ListByUserID(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error)

	// CountByUserID returns the total number of ideas for a user
	CountByUserID(ctx context.Context, userID string) (int64, error)

	// ClearByUserID removes all ideas for a specific user
	// Used when user wants to clear their idea backlog
	ClearByUserID(ctx context.Context, userID string) error

	// DeleteByTopicID removes all ideas for a specific topic
	// Used when a topic is deleted to cascade delete related ideas
	DeleteByTopicID(ctx context.Context, topicID string) error
}
