package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/linkgen-ai/backend/src/domain/entities"
)

// NOTE: These tests are written according to TDD Red pattern - they will FAIL
// until the PromptEngine implementation exists

func TestPromptEngine(t *testing.T) {
	// Test for the PromptEngine service in infrastructure/services:
	// - Cache processed prompts (map[string]string)
	// - Parse and replace variables according to seed/README.md:
	//   * {name} → Topic.name
	//   * {ideas} → Topic.ideas (número)
	//   * {[related_topics]} → strings.Join(Topic.related_topics, ", ")
	//   * {content} → Idea.content
	//   * {user_context} → resultado de buildUserContext() actual
	// - Fallback system for missing prompts

	// Setup
	ctx := context.Background()
	mockRepo := &MockPromptsRepository{}
	engine := NewPromptEngine(mockRepo)

	// Setup test data
	userID := "test-user-123"
	now := time.Now()

	// Setup test user with profile data
	testUser := &entities.User{
		ID:        userID,
		Email:     "test@example.com",
		Industry:  "Technology",
		Role:      "Software Engineer",
		Experience: "5 years",
		Goals:     "Career growth in AI/ML",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Setup test prompts
	base1Prompt := &entities.Prompt{
		ID:             "prompt-1",
		UserID:         userID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Genera {ideas} ideas de contenido únicas y atractivas sobre el siguiente tema: Tema: {name} Temas relacionados: {[related_topics]}",
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	proPrompt := &entities.Prompt{
		ID:             "prompt-2",
		UserID:         userID,
		Name:           "pro",
		Type:           entities.PromptTypeDrafts,
		PromptTemplate: "Escribe un post profesional para LinkedIn sobre el siguiente contenido: Contenido: {content} Contexto del usuario: {user_context}",
		StyleName:      "professional",
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	setup := func() {
		mockRepo.prompts = []*entities.Prompt{base1Prompt, proPrompt}
		engine.ClearCache() // Clear cache for clean tests
	}

	t.Run("should process ideas prompt with variable substitution", func(t *testing.T) {
		setup()

		// GIVEN a topic with related data for ideas prompt
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "Machine Learning for Startups",
			Ideas:           5,
			RelatedTopics:   []string{"AI", "Venture Capital", "Technology Innovation"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing the prompt
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)

		// THEN variables should be replaced correctly
		require.NoError(t, err)
		assert.NotNil(t, processedPrompt)
		
		// Verify variables were substituted
		assert.Contains(t, processedPrompt, "Genera 5 ideas de contenido")
		assert.Contains(t, processedPrompt, "Machine Learning for Startups")
		assert.Contains(t, processedPrompt, "AI, Venture Capital, Technology Innovation")
		assert.NotContains(t, processedPrompt, "{ideas}")
		assert.NotContains(t, processedPrompt, "{name}")
		assert.NotContains(t, processedPrompt, "{[related_topics]}")
	})

	t.Run("should process drafts prompt with content and user context", func(t *testing.T) {
		setup()

		// GIVEN an idea and user for drafts prompt
		idea := &entities.Idea{
			ID:        "idea-1",
			TopicID:   "topic-1",
			Content:   "How ML models are revolutionizing startup growth strategies through predictive analytics",
			Selected:  true,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// WHEN processing the prompt
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, "pro", entities.PromptTypeDrafts, nil, idea, testUser)

		// THEN variables should be replaced correctly
		require.NoError(t, err)
		assert.NotNil(t, processedPrompt)
		
		// Verify content variable was substituted
		assert.Contains(t, processedPrompt, "How ML models are revolutionizing startup growth strategies")
		
		// Verify user context was built and substituted
		assert.Contains(t, processedPrompt, "test@example.com")
		assert.Contains(t, processedPrompt, "Technology")
		assert.Contains(t, processedPrompt, "Software Engineer")
		assert.Contains(t, processedPrompt, "5 years")
		assert.Contains(t, processedPrompt, "Career growth in AI/ML")
		
		assert.NotContains(t, processedPrompt, "{content}")
		assert.NotContains(t, processedPrompt, "{user_context}")
	})

	t.Run("should cache processed prompts", func(t *testing.T) {
		setup()

		// GIVEN a topic
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "Test Topic",
			Ideas:           3,
			RelatedTopics:   []string{"AI"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing the prompt twice
		processed1, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)
		require.NoError(t, err)
		
		processed2, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)
		require.NoError(t, err)

		// THEN results should be identical and cached
		assert.Equal(t, processed1, processed2)
		
		// Verify cache contains the processed prompt
		cacheKey := engine.buildCacheKey(userID, "base1", entities.PromptTypeIdeas, topic, nil)
		cachedValue, exists := engine.GetFromCache(cacheKey)
		assert.True(t, exists)
		assert.Equal(t, processed1, cachedValue)
	})

	t.Run("should use fallback for missing prompts", func(t *testing.T) {
		setup()

		// GIVEN no custom prompt exists for the user
		nonExistentPromptName := "nonexistent-prompt"
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "Test Topic",
			Ideas:           3,
			RelatedTopics:   []string{"AI"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing a non-existent prompt
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, nonExistentPromptName, entities.PromptTypeIdeas, topic, nil, testUser)

		// THEN should fall back to default prompts from seed files
		require.NoError(t, err)
		assert.NotNil(t, processedPrompt)
		
		// Should contain the default prompt content for the type
		assert.Contains(t, processedPrompt, "Genera 3 ideas de contenido")
		assert.Contains(t, processedPrompt, "Test Topic")
		assert.Contains(t, processedPrompt, "AI")
	})

	t.Run("should build user context correctly", func(t *testing.T) {
		setup()

		// WHEN building user context
		userContext := engine.BuildUserContext(testUser)

		// THEN should contain all user profile information
		assert.Contains(t, userContext, "test@example.com")
		assert.Contains(t, userContext, "Technology")
		assert.Contains(t, userContext, "Software Engineer")
		assert.Contains(t, userContext, "5 years")
		assert.Contains(t, userContext, "Career growth in AI/ML")
	})

	t.Run("should handle empty related topics array", func(t *testing.T) {
		setup()

		// GIVEN a topic without related topics
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "Solo Topic",
			Ideas:           2,
			RelatedTopics:   []string{}, // Empty array
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing the prompt
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)

		// THEN should handle empty array correctly
		require.NoError(t, err)
		assert.NotNil(t, processedPrompt)
		assert.Contains(t, processedPrompt, "Genera 2 ideas de contenido")
		assert.Contains(t, processedPrompt, "Solo Topic")
		// Should not include "Temas relacionados:" section when empty
		assert.NotContains(t, processedPrompt, "Temas relacionados:")
	})

	t.Run("should validate required variables before processing", func(t *testing.T) {
		setup()

		// GIVEN a topic with missing name (required for ideas prompt)
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "", // Missing required field
			Ideas:           3,
			RelatedTopics:   []string{"AI"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing the prompt
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)

		// THEN should return validation error
		require.Error(t, err)
		assert.Nil(t, processedPrompt)
		assert.Contains(t, err.Error(), "missing required variable")
	})

	t.Run("should handle drafts prompt fallback for missing style", func(t *testing.T) {
		setup()

		// GIVEN no custom style exists for drafts
		nonExistentStyle := "nonexistent-style"
		idea := &entities.Idea{
			ID:        "idea-1",
			TopicID:   "topic-1",
			Content:   "Content for testing",
			Selected:  true,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// WHEN processing with non-existent style
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, nonExistentStyle, entities.PromptTypeDrafts, nil, idea, testUser)

		// THEN should fall back to default drafts prompt
		require.NoError(t, err)
		assert.NotNil(t, processedPrompt)
		assert.Contains(t, processedPrompt, "Content for testing")
		assert.Contains(t, processedPrompt, "test@example.com")
	})

	t.Run("should return error for invalid user", func(t *testing.T) {
		setup()

		// GIVEN an invalid user (nil)
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "Test Topic",
			Ideas:           3,
			RelatedTopics:   []string{"AI"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing without a valid user
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, nil)

		// THEN should return error
		require.Error(t, err)
		assert.Nil(t, processedPrompt)
		assert.Contains(t, err.Error(), "user is required")
	})

	t.Run("should handle prompt engine initialization", func(t *testing.T) {
		// GIVEN a new prompt engine
		newEngine := NewPromptEngine(mockRepo)

		// WHEN initialized
		// THEN should have empty cache and valid repository reference
		assert.Equal(t, mockRepo, newEngine.GetRepository())
		assert.Empty(t, newEngine.GetCacheContents())
		
		// Should have default prompts loaded from seed
		defaultIdeas := newEngine.GetDefaultPrompt(entities.PromptTypeIdeas)
		assert.NotEmpty(t, defaultIdeas)
		assert.Contains(t, defaultIdeas, "Genera {ideas} ideas")
		
		defaultDrafts := newEngine.GetDefaultPrompt(entities.PromptTypeDrafts)
		assert.NotEmpty(t, defaultDrafts)
		assert.Contains(t, defaultDrafts, "Escribe un post")
	})
}

// Types that don't exist yet (to be implemented)

type PromptEngine struct {
	repository interface{} // Should be interfaces.PromptsRepository
	cache      map[string]string
}

func NewPromptEngine(repo interface{}) *PromptEngine {
	return &PromptEngine{
		repository: repo,
		cache:      make(map[string]string),
	}
}

func (p *PromptEngine) ProcessPrompt(
	ctx context.Context,
	userID string,
	promptName string,
	promptType entities.PromptType,
	topic *entities.Topic,
	idea *entities.Idea,
	user *entities.User,
) (string, error) {
	// TODO: Implementation needed - this test will fail until implemented
	return "", assert.AnError
}

func (p *PromptEngine) BuildUserContext(user *entities.User) string {
	// TODO: Implementation needed - this test will fail until implemented
	return ""
}

func (p *PromptEngine) buildCacheKey(userID string, promptName string, promptType entities.PromptType, topic *entities.Topic, idea *entities.Idea) string {
	// TODO: Implementation needed - this test will fail until implemented
	return ""
}

func (p *PromptEngine) GetFromCache(key string) (string, bool) {
	// TODO: Implementation needed - this test will fail until implemented
	return "", false
}

func (p *PromptEngine) ClearCache() {
	// TODO: Implementation needed - this test will fail until implemented
}

func (p *PromptEngine) GetRepository() interface{} {
	// TODO: Implementation needed - this test will fail until implemented
	return nil
}

func (p *PromptEngine) GetCacheContents() map[string]string {
	// TODO: Implementation needed - this test will fail until implemented
	return nil
}

func (p *PromptEngine) GetDefaultPrompt(promptType entities.PromptType) string {
	// TODO: Implementation needed - this test will fail until implemented
	return ""
}
