package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/factories"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
)

// GenerateIdeasUseCase orchestrates the generation of content ideas
type GenerateIdeasUseCase struct {
	userRepo   interfaces.UserRepository
	topicRepo  interfaces.TopicRepository
	ideasRepo  interfaces.IdeasRepository
	llmService interfaces.LLMService
}

// NewGenerateIdeasUseCase creates a new instance of GenerateIdeasUseCase
func NewGenerateIdeasUseCase(
	userRepo interfaces.UserRepository,
	topicRepo interfaces.TopicRepository,
	ideasRepo interfaces.IdeasRepository,
	llmService interfaces.LLMService,
) *GenerateIdeasUseCase {
	return &GenerateIdeasUseCase{
		userRepo:   userRepo,
		topicRepo:  topicRepo,
		ideasRepo:  ideasRepo,
		llmService: llmService,
	}
}

// GenerateIdeasInput represents input for idea generation
type GenerateIdeasInput struct {
	UserID string
	Count  int
}

const (
	DefaultIdeaCount = 10
	MaxIdeaCount     = 100
	MinIdeaCount     = 1
)

// Execute generates ideas for a user based on a random topic
func (uc *GenerateIdeasUseCase) Execute(ctx context.Context, input GenerateIdeasInput) ([]*entities.Idea, error) {
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

	// Get random topic for user
	topic, err := uc.topicRepo.FindRandomByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find topic: %w", err)
	}
	if topic == nil {
		return nil, fmt.Errorf("no topics configured for user: %s", input.UserID)
	}

	// Call LLM to generate ideas
	ideaContents, err := uc.llmService.GenerateIdeas(ctx, topic.Name, input.Count)
	if err != nil {
		return nil, fmt.Errorf("LLM service error: %w", err)
	}

	if len(ideaContents) == 0 {
		return nil, fmt.Errorf("LLM generated no ideas")
	}

	// Create idea entities using factory
	ideas := make([]*entities.Idea, 0, len(ideaContents))
	for _, content := range ideaContents {
		// Skip empty or whitespace-only ideas
		trimmed := strings.TrimSpace(content)
		if trimmed == "" {
			continue
		}

		idea, err := factories.NewIdea(
			uuid.New().String(),
			input.UserID,
			topic.ID,
			trimmed,
		)
		if err != nil {
			// Log validation error but continue with other ideas
			continue
		}

		ideas = append(ideas, idea)
	}

	if len(ideas) == 0 {
		return nil, fmt.Errorf("no valid ideas could be created from LLM response")
	}

	// Save ideas batch to repository
	if err := uc.ideasRepo.CreateBatch(ctx, ideas); err != nil {
		return nil, fmt.Errorf("failed to save ideas: %w", err)
	}

	return ideas, nil
}

// validateInput validates the input parameters
func (uc *GenerateIdeasUseCase) validateInput(input GenerateIdeasInput) error {
	if strings.TrimSpace(input.UserID) == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if input.Count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}

	if input.Count > MaxIdeaCount {
		return fmt.Errorf("count exceeds maximum allowed (%d)", MaxIdeaCount)
	}

	return nil
}
