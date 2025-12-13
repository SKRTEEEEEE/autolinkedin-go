package services

import (
	"context"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NOTE: These tests are written according to TDD Red pattern - they will FAIL
// until the prompt variable substitution implementation exists

func TestPromptVariableSubstitution(t *testing.T) {
	// This test covers variable substitution in prompt templates:
	// 1. {name} → Topic.name
	// 2. {ideas} → Topic.ideas (number)
	// 3. {[related_topics]} → strings.Join(Topic.related_topics, ", ")
	// 4. {content} → Idea.content
	// 5. {user_context} → generated from user profile/Configuration

	ctx := context.Background()

	// Setup test environment
	testDB := setupTestDB(t, "test_variable_substitution")
	defer testDB.Disconnect(ctx)

	promptsRepo := NewMockPromptsRepository()
	logger := &mockLogger{}
	engine := services.NewPromptEngine(promptsRepo, logger)

	t.Run("should substitute variables in ideas prompts", func(t *testing.T) {
		// Setup test data
		userID := "test-user-123"
		now := time.Now()

		testUser := &entities.User{
			ID:       userID,
			Email:    "test@example.com",
			Language: "es",
			Configuration: map[string]interface{}{
				"name":            "Juan García",
				"expertise":       "Desarrollo Backend",
				"tone_preference": "Profesional",
				"industry":        "Technology",
				"role":            "Software Engineer",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Microservicios en Go",
			Ideas:         3,
			RelatedTopics: []string{"Go", "Arquitectura", "Docker"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		// Create a template with all variables
		template := `Eres un experto en crear contenido para {name}.
Genera exactamente {ideas} ideas sobre el tema.
Considera los temas relacionados: {[related_topics]}.
Usa este contexto: {user_context}`

		// Setup mock repository with this template
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			Name:           "test-ideas",
			StyleName:      "test-ideas",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)

		// Process the prompt
		processed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"test-ideas",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)

		// Verify all variables were substituted
		assert.Contains(t, processed, "Microservicios en Go")
		assert.Contains(t, processed, "3")
		assert.Contains(t, processed, "Go, Arquitectura, Docker")
		assert.Contains(t, processed, "Juan García")
		assert.Contains(t, processed, "Desarrollo Backend")

		// Verify no template variables remain
		assert.NotContains(t, processed, "{name}")
		assert.NotContains(t, processed, "{ideas}")
		assert.NotContains(t, processed, "{[related_topics]}")
		assert.NotContains(t, processed, "{user_context}")
	})

	t.Run("should handle empty related topics in ideas prompts", func(t *testing.T) {
		userID := "test-user-empty"
		now := time.Now()

		testUser := &entities.User{
			ID:        userID,
			Email:     "empty@example.com",
			CreatedAt: now,
			UpdatedAt: now,
		}

		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Simple Topic",
			Ideas:         5,
			RelatedTopics: []string{}, // Empty related topics
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		template := `Genera {ideas} ideas sobre {name}.
Temas relacionados: {[related_topics]}
Más contenido aquí.`

		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			Name:           "empty-related",
			StyleName:      "empty-related",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)

		processed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"empty-related",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)

		// Should handle empty related topics gracefully
		assert.Contains(t, processed, "Simple Topic")
		assert.Contains(t, processed, "5")
		assert.NotContains(t, processed, "{name}")
		assert.NotContains(t, processed, "{ideas}")
		assert.NotContains(t, processed, "{[related_topics]}")

		// Should not contain "Temas relacionados: " with empty value
		assert.NotContains(t, processed, "Temas relacionados: \n")
	})

	t.Run("should substitute variables in draft prompts", func(t *testing.T) {
		userID := "test-draft-user"
		now := time.Now()

		testUser := &entities.User{
			ID:       userID,
			Email:    "draft@example.com",
			Language: "es",
			Configuration: map[string]interface{}{
				"name":            "Maria López",
				"expertise":       "Marketing Digital",
				"tone_preference": "Inspiracional",
				"industry":        "Marketing",
				"role":            "Marketing Manager",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		testIdea := &entities.Idea{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			TopicID:   primitive.NewObjectID().Hex(),
			Content:   "Cómo usar IA para personalizar campañas de marketing y aumentar conversiones",
			Active:    true,
			CreatedAt: now,
			UpdatedAt: now,
		}

		template := `Basado en la idea: {content}

Contexto del usuario:
{user_context}

Instrucciones:
- Crear contenido inspiracional
- Enfocado en marketing digital
- Usar tono profesional`

		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeDrafts,
			Name:           "inspirational",
			StyleName:      "inspiracional",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)

		processed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"inspirational",
			entities.PromptTypeDrafts,
			nil,
			testIdea,
			testUser,
		)
		require.NoError(t, err)

		// Verify variables were substituted
		assert.Contains(t, processed, "Cómo usar IA para personalizar campañas de marketing y aumentar conversiones")
		assert.Contains(t, processed, "Maria López")
		assert.Contains(t, processed, "Marketing Digital")
		assert.Contains(t, processed, "Marketing Manager")

		// Verify no template variables remain
		assert.NotContains(t, processed, "{content}")
		assert.NotContains(t, processed, "{user_context}")
	})

	t.Run("should fall back to legacy fields when Configuration is missing", func(t *testing.T) {
		userID := "legacy-user"
		now := time.Now()

		// User without Configuration (old system)
		testUser := &entities.User{
			ID:    userID,
			Email: "legacy@example.com",
			// No Configuration field
			CreatedAt: now,
			UpdatedAt: now,
		}

		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Test Topic",
			Ideas:         2,
			RelatedTopics: []string{"Topic1"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		template := `Content for {user_context} and {name} with {ideas} ideas.`

		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			Name:           "legacy-test",
			StyleName:      "legacy-test",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)

		processed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"legacy-test",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)

		// Should fall back to email for user context
		assert.Contains(t, processed, "legacy@example.com")
		assert.Contains(t, processed, "Test Topic")
		assert.Contains(t, processed, "2")
		assert.NotContains(t, processed, "{user_context}")
	})

	t.Run("should validate required variables for ideas prompts", func(t *testing.T) {
		userID := "required-fields-user"
		now := time.Now()

		testUser := &entities.User{
			ID:        userID,
			Email:     "required@example.com",
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Topic with empty name
		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "", // Empty name should cause error
			Ideas:         3,
			RelatedTopics: []string{"Topic1", "Topic2"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		template := `Genera {ideas} ideas sobre {name}`

		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			Name:           "required-fields",
			StyleName:      "required-fields",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)

		// Should fail because {name} variable cannot be substituted (empty topic name)
		_, err := engine.ProcessPrompt(
			ctx,
			userID,
			"required-fields",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required variable: {name}")
	})

	t.Run("should validate required variables for draft prompts", func(t *testing.T) {
		userID := "required-draft-user"
		now := time.Now()

		testUser := &entities.User{
			ID:        userID,
			Email:     "draft-required@example.com",
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Idea with empty content
		testIdea := &entities.Idea{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			TopicID:   primitive.NewObjectID().Hex(),
			Content:   "", // Empty content should cause error
			Active:    true,
			CreatedAt: now,
			UpdatedAt: now,
		}

		template := `Basado en: {content}\nContexto: {user_context}`

		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeDrafts,
			Name:           "draft-required",
			StyleName:      "draft-required",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)

		// Should fail because {content} variable cannot be substituted (empty idea content)
		_, err := engine.ProcessPrompt(
			ctx,
			userID,
			"draft-required",
			entities.PromptTypeDrafts,
			nil,
			testIdea,
			testUser,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required variable: {content}")
	})
}

// Mock Prompts Repository for testing
type MockPromptsRepository struct {
	prompts []*entities.Prompt
}

func NewMockPromptsRepository() *MockPromptsRepository {
	return &MockPromptsRepository{
		prompts: make([]*entities.Prompt, 0),
	}
}

func (m *MockPromptsRepository) AddPrompt(prompt *entities.Prompt) {
	m.prompts = append(m.prompts, prompt)
}

func (m *MockPromptsRepository) Create(ctx context.Context, prompt *entities.Prompt) (string, error) {
	m.prompts = append(m.prompts, prompt)
	return prompt.ID, nil
}

func (m *MockPromptsRepository) FindByID(ctx context.Context, id string) (*entities.Prompt, error) {
	for _, prompt := range m.prompts {
		if prompt.ID == id {
			return prompt, nil
		}
	}
	return nil, nil
}

func (m *MockPromptsRepository) FindByName(ctx context.Context, userID string, name string) (*entities.Prompt, error) {
	for _, prompt := range m.prompts {
		if prompt.UserID == userID && prompt.Name == name {
			return prompt, nil
		}
	}
	return nil, nil
}

func (m *MockPromptsRepository) ListByUserID(ctx context.Context, userID string) ([]*entities.Prompt, error) {
	var result []*entities.Prompt
	for _, prompt := range m.prompts {
		if prompt.UserID == userID {
			result = append(result, prompt)
		}
	}
	return result, nil
}

func (m *MockPromptsRepository) ListByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	var result []*entities.Prompt
	for _, prompt := range m.prompts {
		if prompt.UserID == userID && prompt.Type == promptType {
			result = append(result, prompt)
		}
	}
	return result, nil
}

func (m *MockPromptsRepository) FindActiveByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	var result []*entities.Prompt
	for _, prompt := range m.prompts {
		if prompt.UserID == userID && prompt.Type == promptType && prompt.Active {
			result = append(result, prompt)
		}
	}
	return result, nil
}

func (m *MockPromptsRepository) FindByUserIDAndStyle(ctx context.Context, userID string, styleName string) (*entities.Prompt, error) {
	for _, prompt := range m.prompts {
		if prompt.UserID == userID && prompt.StyleName == styleName {
			return prompt, nil
		}
	}
	return nil, nil
}

func (m *MockPromptsRepository) Update(ctx context.Context, prompt *entities.Prompt) error {
	for i, p := range m.prompts {
		if p.ID == prompt.ID {
			m.prompts[i] = prompt
			return nil
		}
	}
	return nil
}

func (m *MockPromptsRepository) Delete(ctx context.Context, id string) error {
	for i, prompt := range m.prompts {
		if prompt.ID == id {
			m.prompts = append(m.prompts[:i], m.prompts[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *MockPromptsRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	count := 0
	for _, prompt := range m.prompts {
		if prompt.UserID == userID {
			count++
		}
	}
	return int64(count), nil
}

// Mock Logger
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
func (m *mockLogger) Info(msg string, fields ...interface{})  {}
func (m *mockLogger) Warn(msg string, fields ...interface{})  {}
func (m *mockLogger) Error(msg string, fields ...interface{}) {}
