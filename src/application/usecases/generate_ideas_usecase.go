package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/factories"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GenerateIdeasUseCase orchestrates the generation of content ideas
type GenerateIdeasUseCase struct {
	userRepo    interfaces.UserRepository
	topicRepo   interfaces.TopicRepository
	ideasRepo   interfaces.IdeasRepository
	promptsRepo interfaces.PromptsRepository
	llmService  interfaces.LLMService
}

// NewGenerateIdeasUseCase creates a new instance of GenerateIdeasUseCase
func NewGenerateIdeasUseCase(
	userRepo interfaces.UserRepository,
	topicRepo interfaces.TopicRepository,
	ideasRepo interfaces.IdeasRepository,
	promptsRepo interfaces.PromptsRepository,
	llmService interfaces.LLMService,
) *GenerateIdeasUseCase {
	return &GenerateIdeasUseCase{
		userRepo:    userRepo,
		topicRepo:   topicRepo,
		ideasRepo:   ideasRepo,
		promptsRepo: promptsRepo,
		llmService:  llmService,
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

// GenerateIdeasForUser generates ideas for a user based on a random topic
func (uc *GenerateIdeasUseCase) GenerateIdeasForUser(ctx context.Context, userID string, count int) ([]*entities.Idea, error) {
	input := GenerateIdeasInput{
		UserID: userID,
		Count:  count,
	}
	return uc.Execute(ctx, input)
}

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

	// Get active ideas prompt for user
	prompts, err := uc.promptsRepo.FindActiveByUserIDAndType(ctx, input.UserID, entities.PromptTypeIdeas)
	if err != nil {
		return nil, fmt.Errorf("failed to find ideas prompt: %w", err)
	}
	if len(prompts) == 0 {
		return nil, fmt.Errorf("no active ideas prompt configured for user: %s", input.UserID)
	}
	
	// Use first active prompt
	prompt := prompts[0]
	
	// Build prompt with variable substitution
	finalPrompt := uc.buildPromptWithVariables(prompt.PromptTemplate, topic, user, input.Count)
	
	// Call LLM with custom prompt
	response, err := uc.llmService.SendRequest(ctx, finalPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM service error: %w", err)
	}
	
	// Parse JSON response
	ideaContents, err := uc.parseIdeasResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
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

		// Generate MongoDB ObjectID
		ideaID := primitive.NewObjectID().Hex()
		
		idea, err := factories.NewIdea(
			ideaID,
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

// buildPromptWithVariables replaces template variables with actual values
func (uc *GenerateIdeasUseCase) buildPromptWithVariables(template string, topic *entities.Topic, user *entities.User, count int) string {
	prompt := template
	
	// Replace {name} with topic name
	prompt = strings.ReplaceAll(prompt, "{name}", topic.Name)
	
	// Replace {related_topics} with topic name (or keywords if available)
	relatedTopics := topic.Name
	if len(topic.Keywords) > 0 {
		relatedTopics = strings.Join(topic.Keywords, ", ")
	}
	prompt = strings.ReplaceAll(prompt, "{related_topics}", relatedTopics)
	
	// Replace {language} with user language
	prompt = strings.ReplaceAll(prompt, "{language}", user.GetLanguage())
	
	// Replace {count} with idea count
	prompt = strings.ReplaceAll(prompt, "{count}", fmt.Sprintf("%d", count))
	
	return prompt
}

// parseIdeasResponse parses the JSON response from LLM
func (uc *GenerateIdeasUseCase) parseIdeasResponse(response string) ([]string, error) {
	var result struct {
		Ideas []string `json:"ideas"`
	}
	
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}
	
	if len(result.Ideas) == 0 {
		return nil, fmt.Errorf("LLM returned empty ideas list")
	}
	
	return result.Ideas, nil
}
