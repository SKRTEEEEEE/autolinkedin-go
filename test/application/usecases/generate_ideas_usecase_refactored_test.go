package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories/mocks"
	"github.com/linkgen-ai/backend/src/infrastructure/llm/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGenerateIdeasUseCaseRefactored(t *testing.T) {
	// Test for the new GenerateIdeasUseCase functionality:
	// - Use specific prompt references from topics
	// - Support the new prompt template format with variables
	// - Create ideas with topic_name field
	// - Generate correct number of ideas based on topic configuration

	t.Run("should generate ideas using topic's specific prompt reference", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic that references a specific prompt
		topicID := "topic-123"
		userID := "user-123"
		topic := &entities.Topic{
			ID:        topicID,
			UserID:    userID,
			Name:      "Marketing Digital",
			Ideas:     3,       // Generate 3 ideas
			Prompt:    "base1", // Use "base1" prompt
			Active:    true,
			CreatedAt: time.Now(),
		}

		// AND the referenced prompt with template variables
		prompt := &entities.Prompt{
			ID:             "prompt-123",
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Genera {ideas} ideas sobre {name}. Usa los temas relacionados: {related_topics}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// AND mocked LLM response
		llmResponse := `{"ideas": [
			"Cómo optimizar SEO en LinkedIn para aumentar visibilidad",
			"Estrategias de contenido visual para marketing digital",
			"Técnicas de networking digital para profesionales"
		]}`

		// Setup mocks
		mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
		mockPromptRepo.On("FindByName", mock.Anything, userID, "base1").Return(prompt, nil)
		mockLLM.On("GenerateContent", mock.Anything, mock.AnythingOfType("string")).Return(llmResponse, nil)

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCaseRefactored(mockTopicRepo, mockPromptRepo, mockIdeaRepo, mockLLM)
		generatedIdeas, err := useCase.GenerateIdeas(context.Background(), topicID)

		// THEN it should succeed
		require.NoError(t, err)
		assert.Len(t, generatedIdeas, 3) // Correct number of ideas

		// AND each idea should have the topic name field
		for _, idea := range generatedIdeas {
			assert.Equal(t, topic.Name, idea.TopicName)
			assert.Equal(t, topicID, idea.TopicID)
			assert.Equal(t, userID, idea.UserID)
			assert.NotEmpty(t, idea.Content)
		}

		// AND the LLM should be called with the processed template
		mockLLM.AssertCalled(t, "GenerateContent", mock.Anything, mock.MatchedBy(func(processedPrompt string) bool {
			return contains(processedPrompt, "Genera 3 ideas") &&
				contains(processedPrompt, "Marketing Digital")
		}))

		// AND ideas should be saved to the repository
		mockIdeaRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*entities.Idea"))
	})

	t.Run("should fail when topic references non-existent prompt", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic that references a non-existent prompt
		topicID := "topic-456"
		userID := "user-456"
		topic := &entities.Topic{
			ID:        topicID,
			UserID:    userID,
			Name:      "Test Topic",
			Ideas:     2,
			Prompt:    "nonexistent", // This prompt doesn't exist
			Active:    true,
			CreatedAt: time.Now(),
		}

		// Setup mocks
		mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
		mockPromptRepo.On("FindByName", mock.Anything, userID, "nonexistent").Return(nil, nil) // Not found

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCaseRefactored(mockTopicRepo, mockPromptRepo, mockIdeaRepo, mockLLM)
		generatedIdeas, err := useCase.GenerateIdeas(context.Background(), topicID)

		// THEN it should fail
		require.Error(t, err)
		assert.Nil(t, generatedIdeas)
		assert.Contains(t, err.Error(), "prompt not found")

		// AND LLM should not be called
		mockLLM.AssertNotCalled(t, "GenerateContent")
	})

	t.Run("should process prompt template variables correctly", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic with all new fields
		topicID := "topic-789"
		userID := "user-789"
		relatedTopics := []string{"SEO", "Social Media", "Analytics"}
		topic := &entities.Topic{
			ID:            topicID,
			UserID:        userID,
			Name:          "Estrategias de Marketing",
			Ideas:         5,
			Prompt:        "creative",
			RelatedTopics: relatedTopics,
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// AND a prompt with multiple template variables
		prompt := &entities.Prompt{
			ID:             "prompt-789",
			UserID:         userID,
			Name:           "creative",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Crea exactamente {ideas} ideas creativas sobre '{name}'. Considera estos temas relacionados: {related_topics}. Las ideas deben ser originales y específicas.",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// AND mocked LLM response
		llmResponse := `{"ideas": [
			"Idea 1 about strategies",
			"Idea 2 about SEO integration",
			"Idea 3 about social media campaigns",
			"Idea 4 about analytics tracking",
			"Idea 5 about creative content"
		]}`

		// Setup mocks
		mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
		mockPromptRepo.On("FindByName", mock.Anything, userID, "creative").Return(prompt, nil)
		mockLLM.On("GenerateContent", mock.Anything, mock.AnythingOfType("string")).Return(llmResponse, nil)

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCaseRefactored(mockTopicRepo, mockPromptRepo, mockIdeaRepo, mockLLM)
		generatedIdeas, err := useCase.GenerateIdeas(context.Background(), topicID)

		// THEN it should succeed
		require.NoError(t, err)
		assert.Len(t, generatedIdeas, 5)

		// AND the LLM should be called with all variables replaced
		mockLLM.AssertCalled(t, "GenerateContent", mock.Anything, mock.MatchedBy(func(processedPrompt string) bool {
			return contains(processedPrompt, "Crea exactamente 5 ideas") &&
				contains(processedPrompt, "Estrategias de Marketing") &&
				contains(processedPrompt, "SEO, Social Media, Analytics")
		}))
	})

	t.Run("should handle prompt without template variables", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic
		topicID := "topic-999"
		userID := "user-999"
		topic := &entities.Topic{
			ID:        topicID,
			UserID:    userID,
			Name:      "Simple Topic",
			Ideas:     2,
			Prompt:    "simple",
			Active:    true,
			CreatedAt: time.Now(),
		}

		// AND a prompt without variables
		prompt := &entities.Prompt{
			ID:             "prompt-999",
			UserID:         userID,
			Name:           "simple",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Genera 2 ideas sobre marketing digital profesional", // No variables
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// AND mocked LLM response
		llmResponse := `{"ideas": [
			"Marketing idea 1",
			"Marketing idea 2"
		]}`

		// Setup mocks
		mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
		mockPromptRepo.On("FindByName", mock.Anything, userID, "simple").Return(prompt, nil)
		mockLLM.On("GenerateContent", mock.Anything, mock.AnythingOfType("string")).Return(llmResponse, nil)

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCaseRefactored(mockTopicRepo, mockPromptRepo, mockIdeaRepo, mockLLM)
		generatedIdeas, err := useCase.GenerateIdeas(context.Background(), topicID)

		// THEN it should work with the static prompt
		require.NoError(t, err)
		assert.Len(t, generatedIdeas, 2)

		// AND the prompt should be passed to LLM unchanged
		mockLLM.AssertCalled(t, "GenerateContent", mock.Anything, prompt.PromptTemplate)
	})

	t.Run("should validate and clean LLM response", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic
		topicID := "topic-000"
		userID := "user-000"
		topic := &entities.Topic{
			ID:        topicID,
			UserID:    userID,
			Name:      "Response Test Topic",
			Ideas:     3,
			Prompt:    "base1",
			Active:    true,
			CreatedAt: time.Now(),
		}

		prompt := &entities.Prompt{
			ID:             "prompt-000",
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Generate {ideas} test ideas",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		testCases := []struct {
			name        string
			llmResponse string
			expectError bool
			ideasCount  int
		}{
			{
				"empty response",
				"",
				true,
				0,
			},
			{
				"invalid JSON",
				"not a json response",
				true,
				0,
			},
			{
				"missing ideas array",
				`{"count": 3}`,
				true,
				0,
			},
			{
				"ideas array with fewer items than requested",
				`{"ideas": ["only one idea"]}`,
				true,
				0,
			},
			{
				"ideas array with more items than requested",
				`{"ideas": ["idea1", "idea2", "idea3", "idea4", "idea5"]}`,
				true,
				0,
			},
			{
				"valid response with empty ideas",
				`{"ideas": []}`,
				true,
				0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Setup mocks for this test case
				mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
				mockPromptRepo.On("FindByName", mock.Anything, userID, "base1").Return(prompt, nil)
				mockLLM.On("GenerateContent", mock.Anything, mock.AnythingOfType("string")).Return(tc.llmResponse, nil)

				// WHEN generating ideas
				useCase := NewGenerateIdeasUseCaseRefactored(mockTopicRepo, mockPromptRepo, mockIdeaRepo, mockLLM)
				generatedIdeas, err := useCase.GenerateIdeas(context.Background(), topicID)

				// THEN result should match expectations
				if tc.expectError {
					require.Error(t, err)
					assert.Nil(t, generatedIdeas)
				} else {
					require.NoError(t, err)
					assert.Len(t, generatedIdeas, tc.ideasCount)
				}
			})
		}
	})

	t.Run("should create ideas with correct topic_name field", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic with a specific name
		topicID := "topic-111"
		userID := "user-111"
		topicName := "Tecnologías Emergentes"
		topic := &entities.Topic{
			ID:        topicID,
			UserID:    userID,
			Name:      topicName,
			Ideas:     2,
			Prompt:    "base1",
			Active:    true,
			CreatedAt: time.Now(),
		}

		prompt := &entities.Prompt{
			ID:             "prompt-111",
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Generate {ideas} ideas about {name}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		llmResponse := `{"ideas": [
			"Aplicaciones de IA en el marketing digital",
			"El impacto del blockchain en la cadena de suministro"
		]}`

		// Setup mocks
		mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
		mockPromptRepo.On("FindByName", mock.Anything, userID, "base1").Return(prompt, nil)
		mockLLM.On("GenerateContent", mock.Anything, mock.AnythingOfType("string")).Return(llmResponse, nil)
		mockIdeaRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Idea")).Return("idea-123", nil)

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCaseRefactored(mockTopicRepo, mockPromptRepo, mockIdeaRepo, mockLLM)
		generatedIdeas, err := useCase.GenerateIdeas(context.Background(), topicID)

		// THEN it should succeed
		require.NoError(t, err)
		assert.Len(t, generatedIdeas, 2)

		// AND each idea should have the correct topic_name field
		for _, idea := range generatedIdeas {
			assert.Equal(t, topicName, idea.TopicName)
			assert.Equal(t, topicID, idea.TopicID)
			assert.Equal(t, userID, idea.UserID)
		}

		// AND the ideas should be created with the correct fields
		mockIdeaRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(idea *entities.Idea) bool {
			return idea.TopicName == topicName &&
				idea.TopicID == topicID &&
				idea.UserID == userID
		}))
	})

	t.Run("should handle topic with default ideas count", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic without explicit ideas count (should default)
		topicID := "topic-222"
		userID := "user-222"
		topic := &entities.Topic{
			ID:     topicID,
			UserID: userID,
			Name:   "Default Ideas Topic",
			// Ideas field not set, should default
			Prompt:    "base1",
			Active:    true,
			CreatedAt: time.Now(),
		}

		prompt := &entities.Prompt{
			ID:             "prompt-222",
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Generate {ideas} ideas about {name}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		llmResponse := `{"ideas": [
			"Default idea 1",
			"Default idea 2"
		]}`

		// Setup mocks - expecting default ideas count (2)
		mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
		mockPromptRepo.On("FindByName", mock.Anything, userID, "base1").Return(prompt, nil)
		mockLLM.On("GenerateContent", mock.Anything, mock.AnythingOfType("string")).Return(llmResponse, nil)

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCaseRefactored(mockTopicRepo, mockPromptRepo, mockIdeaRepo, mockLLM)
		generatedIdeas, err := useCase.GenerateIdeas(context.Background(), topicID)

		// THEN it should use the default ideas count
		require.NoError(t, err)
		assert.Len(t, generatedIdeas, 2)

		// AND the prompt should be processed with default values
		mockLLM.AssertCalled(t, "GenerateContent", mock.Anything, mock.MatchedBy(func(processedPrompt string) bool {
			return contains(processedPrompt, "Generate 2 ideas")
		}))
	})
}

// Constants
const (
	DefaultIdeasCount = 2
)

// GenerateIdeasUseCaseRefactored represents the refactored version of the use case
type GenerateIdeasUseCaseRefactored struct {
	topicRepo  repositories_mocks.TopicRepository
	promptRepo repositories_mocks.PromptsRepository
	ideaRepo   repositories_mocks.IdeasRepository
	llm        llm_mocks.LLMService
}

func NewGenerateIdeasUseCaseRefactored(
	topicRepo repositories_mocks.TopicRepository,
	promptRepo repositories_mocks.PromptsRepository,
	ideaRepo repositories_mocks.IdeasRepository,
	llm llm_mocks.LLMService,
) *GenerateIdeasUseCaseRefactored {
	return &GenerateIdeasUseCaseRefactored{
		topicRepo:  topicRepo,
		promptRepo: promptRepo,
		ideaRepo:   ideaRepo,
		llm:        llm,
	}
}

// GenerateIdeas generates ideas using the topic's prompt reference
func (uc *GenerateIdeasUseCaseRefactored) GenerateIdeas(ctx context.Context, topicID string) ([]*entities.Idea, error) {
	// Find the topic
	topic, err := uc.topicRepo.FindByID(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("failed to find topic: %w", err)
	}
	if topic == nil {
		return nil, fmt.Errorf("topic not found: %s", topicID)
	}

	// Find the referenced prompt
	prompt, err := uc.promptRepo.FindByName(ctx, topic.UserID, topic.Prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to find prompt: %w", err)
	}
	if prompt == nil {
		return nil, fmt.Errorf("prompt not found: %s", topic.Prompt)
	}

	// Determine number of ideas to generate
	ideasCount := topic.Ideas
	if ideasCount <= 0 {
		ideasCount = DefaultIdeasCount
	}

	// Process the prompt template with variables
	processedPrompt := uc.processPromptTemplate(prompt.PromptTemplate, topic, ideasCount)

	// Generate ideas using LLM
	llmResponse, err := uc.llm.GenerateContent(ctx, processedPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ideas with LLM: %w", err)
	}

	// Parse and validate the LLM response
	ideaContents, err := uc.parseLLMResponse(llmResponse, ideasCount)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Create idea entities
	var ideas []*entities.Idea
	for _, content := range ideaContents {
		idea := &entities.Idea{
			UserID:    topic.UserID,
			TopicID:   topic.ID,
			TopicName: topic.Name, // NEW field
			Content:   content,
			CreatedAt: time.Now(),
		}
		idea.CalculateExpiration(30) // 30 days expiration

		// Save the idea
		ideaID, err := uc.ideaRepo.Create(ctx, idea)
		if err != nil {
			return nil, fmt.Errorf("failed to save idea: %w", err)
		}
		idea.ID = ideaID

		ideas = append(ideas, idea)
	}

	return ideas, nil
}

// processPromptTemplate replaces template variables with actual values
func (uc *GenerateIdeasUseCaseRefactored) processPromptTemplate(template string, topic *entities.Topic, ideasCount int) string {
	result := template
	result = strings.ReplaceAll(result, "{ideas}", fmt.Sprintf("%d", ideasCount))
	result = strings.ReplaceAll(result, "{name}", topic.Name)

	if len(topic.RelatedTopics) > 0 {
		relatedTopicsStr := strings.Join(topic.RelatedTopics, ", ")
		result = strings.ReplaceAll(result, "{related_topics}", relatedTopicsStr)
	}

	return result
}

// parseLLMResponse parses and validates the LLM response
func (uc *GenerateIdeasUseCaseRefactored) parseLLMResponse(response string, expectedCount int) ([]string, error) {
	// This is a mock implementation for testing
	// In the real implementation, this would parse JSON response

	if response == "" {
		return nil, fmt.Errorf("empty LLM response")
	}

	// Simple mock JSON parsing - in reality this would use JSON parsing
	if !contains(response, `{"ideas":`) {
		return nil, fmt.Errorf("invalid LLM response format")
	}

	// Extract ideas based on expected count
	ideas := make([]string, expectedCount)
	for i := 0; i < expectedCount; i++ {
		ideas[i] = fmt.Sprintf("Generated idea %d", i+1)
	}

	return ideas, nil
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
