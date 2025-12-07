package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
)

// ListIdeasUseCase orchestrates the listing of ideas for a user
type ListIdeasUseCase struct {
	userRepo  interfaces.UserRepository
	ideasRepo interfaces.IdeasRepository
}

// NewListIdeasUseCase creates a new instance of ListIdeasUseCase
func NewListIdeasUseCase(
	userRepo interfaces.UserRepository,
	ideasRepo interfaces.IdeasRepository,
) *ListIdeasUseCase {
	return &ListIdeasUseCase{
		userRepo:  userRepo,
		ideasRepo: ideasRepo,
	}
}

// ListIdeasInput represents input for listing ideas
type ListIdeasInput struct {
	UserID  string
	TopicID string // Optional: filter by topic
	Limit   int    // Optional: limit results (0 = no limit)
}

const (
	MaxListLimit = 1000
)

// Execute retrieves ideas for a user with optional filters
func (uc *ListIdeasUseCase) Execute(ctx context.Context, input ListIdeasInput) ([]*entities.Idea, error) {
	// Validate input
	if err := uc.validateInput(input); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Verify user exists
	user, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found: %s", input.UserID)
	}

	// Retrieve ideas from repository with filters
	ideas, err := uc.ideasRepo.ListByUserID(ctx, input.UserID, input.TopicID, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve ideas: %w", err)
	}

	// Filter out expired ideas (repository might not do this)
	validIdeas := make([]*entities.Idea, 0, len(ideas))
	for _, idea := range ideas {
		// Include all ideas, even expired ones, for user review
		// The user might want to see what expired
		validIdeas = append(validIdeas, idea)
	}

	return validIdeas, nil
}

// validateInput validates the input parameters
func (uc *ListIdeasUseCase) validateInput(input ListIdeasInput) error {
	if strings.TrimSpace(input.UserID) == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if input.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}

	if input.Limit > MaxListLimit {
		return fmt.Errorf("limit exceeds maximum allowed (%d)", MaxListLimit)
	}

	return nil
}
