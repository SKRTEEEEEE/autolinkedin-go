package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories/mocks"
	"github.com/linkgen-ai/backend/src/infrastructure/llm/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestGenerateIdeasUseCaseFase1 tests the refactored GenerateIdeasUseCase for Phase 1
// TDD Red: Tests will fail initially as the refactored code doesn't exist yet
func TestGenerateIdeasUseCaseFase1(t *testing.T) {

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

		// AND the referenced prompt from seed/prompt/base1.idea.md
		prompt := &entities.Prompt{
			ID:     "prompt-123",
			UserID: userID,
			Name:   "base1",
			Type:   entities.PromptTypeIdeas,
			PromptTemplate: `Eres un experto en estrategia de contenido para LinkedIn. Genera {ideas} ideas de contenido únicas y atractivas sobre el siguiente tema:

Tema: {name}
Temas relacionados: {related_topics}

Requisitos:
- Cada idea debe ser específica y accionable
- Las ideas deben ser diversas y cubrir diferentes ángulos
- Enfócate en valor profesional e insights
- Mantén las ideas concisas (1-2 oraciones cada una)
- Hazlas adecuadas para la audiencia de LinkedIn
- IMPORTANTE: Genera el contenido SIEMPRE en español

Devuelve ÚNICAMENTE un objeto JSON con este formato exacto:
{"ideas": ["idea1", "idea2", "idea3", ...]}`,
			Active:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
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
		mockIdeaRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*entities.Idea")).Return(nil)
		mockLLM.On("SendRequest", mock.Anything, mock.AnythingOfType("string")).Return(llmResponse, nil)

		// WHEN generating ideas using the refactored use case
		useCase := NewGenerateIdeasUseCase(
			nil, // userRepo
			mockTopicRepo,
			mockIdeaRepo,
			mockPromptRepo,
			mockLLM,
		)

		// This will fail because the refactored Execute method doesn't exist yet
		generatedIdeas, err := useCase.GenerateIdeasForTopic(context.Background(), topicID)

		// THEN it should eventually succeed
		require.NoError(t, err)
		assert.Len(t, generatedIdeas, 3)

		// AND each idea should have the topic_name field with correct value
		for _, idea := range generatedIdeas {
			assert.Equal(t, topic.Name, idea.TopicName)
			assert.Equal(t, topicID, idea.TopicID)
			assert.Equal(t, userID, idea.UserID)
			assert.NotEmpty(t, idea.Content)
			assert.False(t, idea.Used)
		}

		// AND the LLM should be called with the processed template
		mockLLM.AssertCalled(t, "SendRequest", mock.Anything, mock.MatchedBy(func(processedPrompt string) bool {
			return strings.Contains(processedPrompt, "Genera 3 ideas") &&
				strings.Contains(processedPrompt, "Marketing Digital")
		}))
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
		mockPromptRepo.On("FindByName", mock.Anything, userID, "nonexistent").Return(nil, nil)

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCase(
			nil,
			mockTopicRepo,
			mockIdeaRepo,
			mockPromptRepo,
			mockLLM,
		)
		_, err := useCase.GenerateIdeasForTopic(context.Background(), topicID)

		// THEN it should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt not found")

		// AND LLM should not be called
		mockLLM.AssertNotCalled(t, "SendRequest")
	})

	t.Run("should use topic's ideas field to determine count", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic specifying 5 ideas
		topicID := "topic-456"
		userID := "user-456"
		topic := &entities.Topic{
			ID:        topicID,
			UserID:    userID,
			Name:      "SEO Strategy",
			Ideas:     5, // Request 5 ideas
			Prompt:    "base1",
			Active:    true,
			CreatedAt: time.Now(),
		}

		prompt := &entities.Prompt{
			ID:             "prompt-456",
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Generate {ideas} ideas about {name}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		llmResponse := `{"ideas": [
			"Idea 1",
			"Idea 2",
			"Idea 3",
			"Idea 4",
			"Idea 5"
		]}`

		// Setup mocks
		mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
		mockPromptRepo.On("FindByName", mock.Anything, userID, "base1").Return(prompt, nil)
		mockIdeaRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*entities.Idea")).Return(nil)
		mockLLM.On("SendRequest", mock.Anything, mock.AnythingOfType("string")).Return(llmResponse, nil)

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCase(
			nil,
			mockTopicRepo,
			mockIdeaRepo,
			mockPromptRepo,
			mockLLM,
		)
		ideas, err := useCase.GenerateIdeasForTopic(context.Background(), topicID)

		// THEN it should generate exactly 5 ideas
		require.NoError(t, err)
		assert.Len(t, ideas, 5)

		// AND prompt should be processed with correct count
		mockLLM.AssertCalled(t, "SendRequest", mock.Anything, mock.MatchedBy(func(processedPrompt string) bool {
			return strings.Contains(processedPrompt, "Genera 5 ideas")
		}))
	})

	t.Run("should handle topic with related_topics field", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic with related topics
		topicID := "topic-789"
		userID := "user-789"
		relatedTopics := []string{"SEO", "Social Media", "Analytics"}
		topic := &entities.Topic{
			ID:            topicID,
			UserID:        userID,
			Name:          "Digital Marketing",
			Ideas:         3,
			Prompt:        "base1",
			RelatedTopics: relatedTopics,
			Active:        true,
			CreatedAt:     time.Now(),
		}

		prompt := &entities.Prompt{
			ID:             "prompt-789",
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: `Genera {ideas} ideas sobre {name}. Temas relacionados: {related_topics}`,
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		llmResponse := `{"ideas": ["Idea 1", "Idea 2", "Idea 3"]}`

		// Setup mocks
		mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
		mockPromptRepo.On("FindByName", mock.Anything, userID, "base1").Return(prompt, nil)
		mockIdeaRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*entities.Idea")).Return(nil)
		mockLLM.On("SendRequest", mock.Anything, mock.AnythingOfType("string")).Return(llmResponse, nil)

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCase(
			nil,
			mockTopicRepo,
			mockIdeaRepo,
			mockPromptRepo,
			mockLLM,
		)
		ideas, err := useCase.GenerateIdeasForTopic(context.Background(), topicID)

		// THEN it should succeed
		require.NoError(t, err)
		assert.Len(t, ideas, 3)

		// AND prompt should include related topics
		mockLLM.AssertCalled(t, "SendRequest", mock.Anything, mock.MatchedBy(func(processedPrompt string) bool {
			return strings.Contains(processedPrompt, "Temas relacionados: SEO, Social Media, Analytics")
		}))
	})

	t.Run("should validate idea content length (max 200 chars)", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic
		topicID := "topic-200"
		userID := "user-200"
		topic := &entities.Topic{
			ID:        topicID,
			UserID:    userID,
			Name:      "Content Length Test",
			Ideas:     1,
			Prompt:    "base1",
			Active:    true,
			CreatedAt: time.Now(),
		}

		prompt := &entities.Prompt{
			ID:             "prompt-200",
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Generate {ideas} test ideas",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Generate an idea longer than 200 characters
		longIdea := strings.Repeat("a", 201)
		llmResponse := fmt.Sprintf(`{"ideas": ["%s"]}`, longIdea)

		// Setup mocks
		mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
		mockPromptRepo.On("FindByName", mock.Anything, userID, "base1").Return(prompt, nil)
		mockLLM.On("SendRequest", mock.Anything, mock.AnythingOfType("string")).Return(llmResponse, nil)

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCase(
			nil,
			mockTopicRepo,
			mockIdeaRepo,
			mockPromptRepo,
			mockLLM,
		)
		ideas, err := useCase.GenerateIdeasForTopic(context.Background(), topicID)

		// THEN it should fail due to content length validation
		if err == nil {
			t.Error("Expected idea generation to fail with content too long")
		}
		assert.Contains(t, err.Error(), "content too long")
		assert.Nil(t, ideas)
	})

	t.Run("should save ideas with topic_name field", func(t *testing.T) {
		// GIVEN mocked dependencies
		mockTopicRepo := new(repositories_mocks.TopicRepository)
		mockPromptRepo := new(repositories_mocks.PromptsRepository)
		mockIdeaRepo := new(repositories_mocks.IdeasRepository)
		mockLLM := new(llm_mocks.LLMService)

		// AND a topic with a specific name
		topicID := "topic-save"
		userID := "user-save"
		topicName := "Professional Development"
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
			ID:             "prompt-save",
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Generate {ideas} ideas about {name}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		llmResponse := `{"ideas": [
			"Build leadership skills through practice",
			"Develop continuous learning habits"
		]}`

		// Setup mocks
		mockTopicRepo.On("FindByID", mock.Anything, topicID).Return(topic, nil)
		mockPromptRepo.On("FindByName", mock.Anything, userID, "base1").Return(prompt, nil)
		mockIdeaRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*entities.Idea")).Return(nil).Run(func(args mock.Arguments) {
			ideas := args.Get(1).([]*entities.Idea)
			// Verify topic_name is set on each idea
			for _, idea := range ideas {
				if idea.TopicName != topicName {
					t.Errorf("Expected topic_name to be '%s', got '%s'", topicName, idea.TopicName)
				}
			}
		})
		mockLLM.On("SendRequest", mock.Anything, mock.AnythingOfType("string")).Return(llmResponse, nil)

		// WHEN generating ideas
		useCase := NewGenerateIdeasUseCase(
			nil,
			mockTopicRepo,
			mockIdeaRepo,
			mockPromptRepo,
			mockLLM,
		)
		ideas, err := useCase.GenerateIdeasForTopic(context.Background(), topicID)

		// THEN it should succeed
		require.NoError(t, err)
		assert.Len(t, ideas, 2)

		// AND CreateBatch should be called
		mockIdeaRepo.AssertCalled(t, "CreateBatch", mock.Anything, mock.AnythingOfType("[]*entities.Idea"))
	})
}

// GenerateIdeasForTopic is a new method that doesn't exist yet in GenerateIdeasUseCase
// This test will fail until the method is implemented
func (uc *GenerateIdeasUseCase) GenerateIdeasForTopic(ctx context.Context, topicID string) ([]*entities.Idea, error) {
	// This is the new method that will be implemented
	// For now, it doesn't exist, so the test will fail
	return nil, fmt.Errorf("method GenerateIdeasForTopic not implemented yet")
}

// Helper function to validate that the refactored use case will use prompts from seed/prompt/
func TestSeedPromptTemplateParsing(t *testing.T) {
	t.Run("should parse prompt template from seed/prompt/base1.idea.md", func(t *testing.T) {
		// GIVEN the content of base1.idea.md
		promptTemplate := `Eres un experto en estrategia de contenido para LinkedIn. Genera {ideas} ideas de contenido únicas y atractivas sobre el siguiente tema:

Tema: {name}
Temas relacionados: {related_topics}

Requisitos:
- Cada idea debe ser específica y accionable
- Las ideas deben ser diversas y cubrir diferentes ángulos
- Enfócate en valor profesional e insights
- Mantén las ideas concisas (1-2 oraciones cada una)
- Hazlas adecuadas para la audiencia de LinkedIn
- IMPORTANTE: Genera el contenido SIEMPRE en español

Devuelve ÚNICAMENTE un objeto JSON con este formato exacto:
{"ideas": ["idea1", "idea2", "idea3", ...]}`

		// AND values to substitute
		values := map[string]string{
			"ideas":          "3",
			"name":           "Marketing Digital",
			"related_topics": "SEO, Social Media",
		}

		// WHEN processing the template
		processed := processTemplateVariables(promptTemplate, values)

		// THEN variables should be replaced
		assert.Contains(t, processed, "Genera 3 ideas")
		assert.Contains(t, processed, "Tema: Marketing Digital")
		assert.Contains(t, processed, "Temas relacionados: SEO, Social Media")

		// AND template placeholders should not remain
		assert.NotContains(t, processed, "{ideas}")
		assert.NotContains(t, processed, "{name}")
		assert.NotContains(t, processed, "{related_topics}")
	})
}

// processTemplateVariables is a helper function that will be implemented
func processTemplateVariables(template string, values map[string]string) string {
	result := template
	for key, value := range values {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// Test for new Topic entity structure
func TestTopicEntityFase1(t *testing.T) {
	t.Run("should validate new Topic fields", func(t *testing.T) {
		// GIVEN a topic with all new fields
		topic := &entities.Topic{
			ID:            "topic-123",
			UserID:        "user-123",
			Name:          "Marketing Strategy",
			Ideas:         5,       // New field
			Prompt:        "base1", // New field
			RelatedTopics: []string{"SEO", "Content"},
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// WHEN validating
		err := topic.Validate()

		// THEN it should be valid (when Validate method is updated)
		if err != nil {
			t.Errorf("Expected topic with new fields to be valid, got error: %v", err)
		}

		// AND new fields should be accessible
		assert.Equal(t, 5, topic.Ideas)
		assert.Equal(t, "base1", topic.Prompt)
		assert.Equal(t, []string{"SEO", "Content"}, topic.RelatedTopics)
	})
}

// Test for new Idea entity structure with reduced content length
func TestIdeaEntityFase1(t *testing.T) {
	t.Run("should enforce 200 character limit on content", func(t *testing.T) {
		// GIVEN an idea with exactly 200 characters
		validIdea := strings.Repeat("a", 200)
		idea := &entities.Idea{
			ID:        "idea-123",
			UserID:    "user-123",
			TopicID:   "topic-123",
			TopicName: "Test Topic",
			Content:   validIdea,
			CreatedAt: time.Now(),
		}

		// WHEN validating
		err := idea.Validate()

		// THEN it should be valid
		if err != nil {
			t.Errorf("Expected idea with 200 characters to be valid, got error: %v", err)
		}

		// WHEN content exceeds 200 characters
		idea.Content = strings.Repeat("b", 201)
		err = idea.Validate()

		// THEN it should be invalid
		if err == nil {
			t.Error("Expected idea with 201 characters to be invalid")
		}
		assert.Contains(t, err.Error(), "too long")
	})

	t.Run("should require topic_name field", func(t *testing.T) {
		// GIVEN an idea without topic_name
		idea := &entities.Idea{
			ID:        "idea-456",
			UserID:    "user-456",
			TopicID:   "topic-456",
			TopicName: "", // Missing field
			Content:   "Valid content",
			CreatedAt: time.Now(),
		}

		// WHEN validating
		err := idea.Validate()

		// THEN it should fail
		if err == nil {
			t.Error("Expected idea without topic_name to be invalid")
		}

		// WHEN topic_name is provided
		idea.TopicName = "Test Topic"
		err = idea.Validate()

		// THEN it should be valid (when Validate method is updated)
		if err != nil && !strings.Contains(err.Error(), "topic name") {
			t.Errorf("Unexpected validation error: %v", err)
		}
	})
}
