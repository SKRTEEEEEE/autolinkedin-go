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
		Count:  uc.determineIdeaCount(count),
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

	return uc.generateIdeasForTopicContext(ctx, topic, user)
}

// GenerateIdeasForTopic generates ideas for a specific topic by ID
func (uc *GenerateIdeasUseCase) GenerateIdeasForTopic(ctx context.Context, topicID string) ([]*entities.Idea, error) {
	// Validate topic ID
	if strings.TrimSpace(topicID) == "" {
		return nil, fmt.Errorf("topic ID cannot be empty")
	}

	// Find topic by ID
	topic, err := uc.topicRepo.FindByID(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("failed to find topic: %w", err)
	}
	if topic == nil {
		return nil, fmt.Errorf("topic not found: %s", topicID)
	}

	// Verify user exists
	user, err := uc.userRepo.FindByID(ctx, topic.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found: %s", topic.UserID)
	}

	return uc.generateIdeasForTopicContext(ctx, topic, user)
}

func (uc *GenerateIdeasUseCase) generateIdeasForTopicContext(ctx context.Context, topic *entities.Topic, user *entities.User) ([]*entities.Idea, error) {
	if topic == nil {
		return nil, fmt.Errorf("topic cannot be nil")
	}

	if user == nil {
		return nil, fmt.Errorf("user cannot be nil")
	}

	prompt, err := uc.resolvePrompt(ctx, topic)
	if err != nil {
		return nil, err
	}

	ideaCount := uc.determineIdeaCount(topic.Ideas)
	finalPrompt := uc.buildPromptWithVariablesFromTopic(prompt.PromptTemplate, topic, user, ideaCount)

	ideaContents, err := uc.requestIdeasFromLLM(ctx, finalPrompt)
	if err != nil {
		return nil, err
	}

	limitedIdeas := ideaContents
	if ideaCount > 0 && len(ideaContents) > ideaCount {
		limitedIdeas = ideaContents[:ideaCount]
	}

	ideas := make([]*entities.Idea, 0, len(limitedIdeas))
	for _, content := range limitedIdeas {
		trimmed := uc.sanitizeIdeaContent(content)
		if trimmed == "" {
			continue
		}

		ideaID := primitive.NewObjectID().Hex()
		idea, err := factories.NewIdea(
			ideaID,
			topic.UserID,
			topic.ID,
			topic.Name,
			trimmed,
		)
		if err != nil {
			continue
		}

		ideas = append(ideas, idea)
	}

	if len(ideas) == 0 {
		return nil, fmt.Errorf("no valid ideas could be created from LLM response")
	}

	if err := uc.ideasRepo.CreateBatch(ctx, ideas); err != nil {
		return nil, fmt.Errorf("failed to save ideas: %w", err)
	}

	return ideas, nil
}

func (uc *GenerateIdeasUseCase) resolvePrompt(ctx context.Context, topic *entities.Topic) (*entities.Prompt, error) {
	if topic == nil {
		return nil, fmt.Errorf("topic cannot be nil")
	}

	var prompt *entities.Prompt

	if topic.Prompt != "" {
		foundPrompt, err := uc.promptsRepo.FindByName(ctx, topic.UserID, strings.TrimSpace(topic.Prompt))
		if err != nil {
			return nil, fmt.Errorf("failed to find prompt: %w", err)
		}

		if foundPrompt != nil {
			prompt = foundPrompt
		}
	}

	if prompt == nil {
		fallbackPrompts, err := uc.promptsRepo.FindActiveByUserIDAndType(ctx, topic.UserID, entities.PromptTypeIdeas)
		if err != nil {
			return nil, fmt.Errorf("failed to find fallback prompt: %w", err)
		}
		if len(fallbackPrompts) == 0 {
			return nil, fmt.Errorf("no active prompts available for ideas")
		}
		prompt = fallbackPrompts[0]
	}

	if prompt.Type != entities.PromptTypeIdeas {
		return nil, fmt.Errorf("prompt is not of type ideas: %s", prompt.Name)
	}

	return prompt, nil
}

func (uc *GenerateIdeasUseCase) requestIdeasFromLLM(ctx context.Context, prompt string) ([]string, error) {
	response, err := uc.llmService.SendRequest(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM service error: %w", err)
	}

	ideaContents, err := uc.parseIdeasResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(ideaContents) == 0 {
		return nil, fmt.Errorf("LLM generated no ideas")
	}

	return ideaContents, nil
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
	return uc.replaceTemplateVariables(template, topic, user, count)
}

// buildPromptWithVariablesFromTopic replaces template variables with actual values for a specific topic
func (uc *GenerateIdeasUseCase) buildPromptWithVariablesFromTopic(template string, topic *entities.Topic, user *entities.User, count int) string {
	return uc.replaceTemplateVariables(template, topic, user, count)
}

// replaceTemplateVariables performs variable substitution for idea prompts.
func (uc *GenerateIdeasUseCase) replaceTemplateVariables(template string, topic *entities.Topic, user *entities.User, count int) string {
	relatedTopics := strings.Join(topic.RelatedTopics, ", ")
	if strings.TrimSpace(relatedTopics) == "" {
		relatedTopics = ""
	}

	replacers := []string{
		"{name}", topic.Name,
		"{topic_name}", topic.Name,
		"{topic}", topic.Name,
		"{topic_description}", topic.Description,
		"{ideas}", fmt.Sprintf("%d", count),
		"{count}", fmt.Sprintf("%d", count),
		"{language}", user.GetLanguage(),
		"{related_topics}", relatedTopics,
		"{[related_topics]}", relatedTopics,
	}

	replacer := strings.NewReplacer(replacers...)
	prompt := replacer.Replace(template)

	// Clean up double spaces when optional values are empty
	return strings.TrimSpace(prompt)
}

// determineIdeaCount enforces defaults and minimums for requested idea count.
func (uc *GenerateIdeasUseCase) determineIdeaCount(count int) int {
	if count <= 0 {
		return entities.DefaultIdeasCount
	}

	if count > entities.MaxIdeasCount {
		return entities.MaxIdeasCount
	}

	return count
}

// sanitizeIdeaContent trims whitespace and enforces maximum length limits.
func (uc *GenerateIdeasUseCase) sanitizeIdeaContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}

	if len(trimmed) > entities.MaxIdeaContentLength {
		trimmed = strings.TrimSpace(trimmed[:entities.MaxIdeaContentLength])
	}

	if len(trimmed) < entities.MinIdeaContentLength {
		padding := entities.MinIdeaContentLength - len(trimmed)
		trimmed = fmt.Sprintf("%s%s", trimmed, strings.Repeat(".", padding))
	}

	return trimmed
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
