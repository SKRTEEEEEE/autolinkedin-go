package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/linkgen-ai/backend/src/domain/interfaces"
)

// ClearIdeasUseCase orchestrates clearing of accumulated ideas
type ClearIdeasUseCase struct {
	userRepo  interfaces.UserRepository
	ideasRepo interfaces.IdeasRepository
}

// NewClearIdeasUseCase creates a new instance of ClearIdeasUseCase
func NewClearIdeasUseCase(
	userRepo interfaces.UserRepository,
	ideasRepo interfaces.IdeasRepository,
) *ClearIdeasUseCase {
	return &ClearIdeasUseCase{
		userRepo:  userRepo,
		ideasRepo: ideasRepo,
	}
}

// ClearIdeasInput represents input for clearing ideas
type ClearIdeasInput struct {
	UserID string
}

// ClearIdeasResult represents the result of clearing ideas
type ClearIdeasResult struct {
	DeletedCount int64
}

// Execute removes all ideas for a user
func (uc *ClearIdeasUseCase) Execute(ctx context.Context, input ClearIdeasInput) (*ClearIdeasResult, error) {
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

	// Count existing ideas before deletion
	count, err := uc.ideasRepo.CountByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to count ideas: %w", err)
	}

	// Clear all ideas for user
	if err := uc.ideasRepo.ClearByUserID(ctx, input.UserID); err != nil {
		return nil, fmt.Errorf("failed to clear ideas: %w", err)
	}

	return &ClearIdeasResult{
		DeletedCount: count,
	}, nil
}

// validateInput validates the input parameters
func (uc *ClearIdeasUseCase) validateInput(input ClearIdeasInput) error {
	if strings.TrimSpace(input.UserID) == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	return nil
}
