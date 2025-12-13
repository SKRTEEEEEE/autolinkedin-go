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
// until the prompt fallback implementation exists

func TestPromptFallback(t *testing.T) {
	// This test covers fallback behavior when custom prompts are missing:
	// 1. Use default prompts when custom prompt not found
	// 2. Verify default prompts contain all required variables
	// 3. Verify default prompts work with variable substitution
	// 4. Test different default prompts for ideas vs drafts

	ctx := context.Background()

	// Setup test environment
	testDB := setupTestDB(t, "test_fallback")
	defer testDB.Disconnect(ctx)

	promptsRepo := NewMockPromptsRepository()
	logger := &mockLogger{}
	engine := services.NewPromptEngine(promptsRepo, logger)

	t.Run("should fallback to default ideas prompt when custom not found", func(t *testing.T) {
		// Setup test data
		userID := "fallback-ideas-user"
		now := time.Now()

		testUser := &entities.User{
			ID:    userID,
			Email: "fallback-ideas@example.com",
			Configuration: map[string]interface{}{
				"name": "Ana Martínez",
				"role": "Product Manager",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Gestión de Ágil en equipos remotos",
			Ideas:         4,
			RelatedTopics: []string{"Agile", "Remote Work", "Team Management"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		// Don't add any custom prompts to repository - should force fallback
		processed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent-prompt", // This doesn't exist
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)

		// Should contain default content with variables substituted
		assert.Contains(t, processed, "Gestión de Ágil en equipos remotos")  // {name}
		assert.Contains(t, processed, "4")                                   // {ideas}
		assert.Contains(t, processed, "Agile, Remote Work, Team Management") // {[related_topics]}
		assert.Contains(t, processed, "Ana Martínez")                        // {user_context}

		// Should contain default Spanish instruction
		assert.Contains(t, processed, "experto en estrategia de contenido para LinkedIn")
		assert.Contains(t, processed, "Genera 4 ideas") // Variable was substituted
		assert.Contains(t, processed, "json con este formato")

		// Should not contain template variables
		assert.NotContains(t, processed, "{name}")
		assert.NotContains(t, processed, "{ideas}")
		assert.NotContains(t, processed, "{[related_topics]}")
		assert.NotContains(t, processed, "{user_context}")
	})

	t.Run("should fallback to default drafts prompt when custom not found", func(t *testing.T) {
		userID := "fallback-drafts-user"
		now := time.Now()

		testUser := &entities.User{
			ID:       userID,
			Email:    "fallback-drafts@example.com",
			Language: "es",
			Configuration: map[string]interface{}{
				"name": "Carlos Rodríguez",
				"role": "CTO",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		testIdea := &entities.Idea{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			TopicID:   primitive.NewObjectID().Hex(),
			Content:   "Cómo implementar microservicios con tolerancia a fallos usando circuit breakers",
			Active:    true,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Don't add any custom prompts to repository - should force fallback
		processed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent-style", // This doesn't exist
			entities.PromptTypeDrafts,
			nil,
			testIdea,
			testUser,
		)
		require.NoError(t, err)

		// Should contain default content with variables substituted
		assert.Contains(t, processed, "Cómo implementar microservicios con tolerancia a fallos usando circuit breakers") // {content}
		assert.Contains(t, processed, "Carlos Rodríguez")                                                                // {user_context}

		// Should contain default Spanish instruction for drafts
		assert.Contains(t, processed, "experto creador de contenido para LinkedIn")
		assert.Contains(t, processed, "Basándote en la siguiente idea")
		assert.Contains(t, processed, "Contexto adicional del usuario")
		assert.Contains(t, processed, "español neutro profesional")
		assert.Contains(t, processed, "FORMATO OBLIGATORIO")
		assert.Contains(t, processed, "posts")    // Default posts array
		assert.Contains(t, processed, "articles") // Default articles array

		// Should not contain template variables
		assert.NotContains(t, processed, "{content}")
		assert.NotContains(t, processed, "{user_context}")
	})

	t.Run("should handle fallback with minimal user data", func(t *testing.T) {
		userID := "minimal-fallback-user"
		now := time.Now()

		// User with minimal data
		testUser := &entities.User{
			ID:    userID,
			Email: "minimal@example.com",
			// No Configuration field
		}

		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Simple Topic",
			Ideas:         3,
			RelatedTopics: []string{}, // Empty array
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		processed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)

		// Should still work with minimal data
		assert.Contains(t, processed, "Simple Topic")
		assert.Contains(t, processed, "3")
		assert.Contains(t, processed, "minimal@example.com") // Fallback context uses email

		// Should handle empty related topics gracefully
		assert.NotContains(t, processed, "{[related_topics]}")
	})

	t.Run("should use custom prompt when available and fallback only when missing", func(t *testing.T) {
		userID := "mixed-fallback-user"
		now := time.Now()

		testUser := &entities.User{
			ID:        userID,
			Email:     "mixed@example.com",
			CreatedAt: now,
			UpdatedAt: now,
		}

		testTopic1 := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Custom Topic",
			Ideas:         5,
			RelatedTopics: []string{"Custom"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		testTopic2 := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Default Topic",
			Ideas:         3,
			RelatedTopics: []string{"Default"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		// Create custom prompt for "custom" name
		customPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			Name:           "custom",
			StyleName:      "custom",
			PromptTemplate: "CUSTOM TEMPLATE: Generate {ideas} creative ideas about {name} including {[related_topics]}",
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(customPrompt)

		// Process with custom prompt (should use custom, not fallback)
		customProcessed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"custom", // This exists
			entities.PromptTypeIdeas,
			testTopic1,
			nil,
			testUser,
		)
		require.NoError(t, err)

		assert.Contains(t, customProcessed, "CUSTOM TEMPLATE")          // Custom content
		assert.Contains(t, customProcessed, "5")                        // Variable substituted
		assert.Contains(t, customProcessed, "Custom Topic")             // Variable substituted
		assert.NotContains(t, customProcessed, "experto en estrategia") // Not default

		// Process with non-existent prompt (should fallback)
		defaultProcessed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent", // This doesn't exist
			entities.PromptTypeIdeas,
			testTopic2,
			nil,
			testUser,
		)
		require.NoError(t, err)

		assert.Contains(t, defaultProcessed, "experto en estrategia") // Default content
		assert.Contains(t, defaultProcessed, "3")                     // Variable substituted
		assert.Contains(t, defaultProcessed, "Default Topic")         // Variable substituted
		assert.NotContains(t, defaultProcessed, "CUSTOM TEMPLATE")    // Not custom
	})

	t.Run("should return error when default prompt not available", func(t *testing.T) {
		userID := "error-fallback-user"
		now := time.Now()

		testUser := &entities.User{
			ID:        userID,
			Email:     "error@example.com",
			CreatedAt: now,
			UpdatedAt: now,
		}

		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Error Topic",
			Ideas:         3,
			RelatedTopics: []string{"Error"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		// Try to process with invalid prompt type that has no default
		// Note: This would require the GetDefaultPrompt method to return empty string for invalid types
		// The test will fail if the implementation doesn't handle this case properly
		processed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent",
			entities.PromptType("invalid-type"), // Invalid type with no default
			testTopic,
			nil,
			testUser,
		)

		// Should return error instead of panic
		if len(processed) == 0 {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no default prompt found")
		}
	})

	t.Run("should verify all default required variables exist", func(t *testing.T) {
		userID := "verify-variables-user"
		now := time.Now()

		testUser := &entities.User{
			ID:        userID,
			Email:     "verify@example.com",
			CreatedAt: now,
			UpdatedAt: now,
		}

		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Variable Test Topic",
			Ideas:         2,
			RelatedTopics: []string{"Variables"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		testIdea := &entities.Idea{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			Content:   "Test idea content for variable verification",
			Active:    true,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Test default ideas prompt has all expected variables
		ideasPrompt, err := engine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)

		// Check all expected variables were in the original template
		originalDefault := engine.GetDefaultPrompt(entities.PromptTypeIdeas)
		assert.Contains(t, originalDefault, "{name}")
		assert.Contains(t, originalDefault, "{ideas}")
		assert.Contains(t, originalDefault, "{[related_topics]}")

		// Test default drafts prompt has all expected variables
		draftsPrompt, err := engine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent",
			entities.PromptTypeDrafts,
			nil,
			testIdea,
			testUser,
		)
		require.NoError(t, err)

		// Check all expected variables were in the original template
		originalDraft := engine.GetDefaultPrompt(entities.PromptTypeDrafts)
		assert.Contains(t, originalDraft, "{content}")
		assert.Contains(t, originalDraft, "{user_context}")
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
