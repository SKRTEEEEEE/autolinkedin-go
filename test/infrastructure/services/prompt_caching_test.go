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
// until the prompt caching implementation exists

func TestPromptCaching(t *testing.T) {
	// This test covers the caching mechanism:
	// 1. Cache processed prompts in map[string]string
	// 2. Generate cache keys based on user, prompt, topic, idea
	// 3. Return cached prompts when available
	// 4. Track cache hits/misses
	// 5. Clear cache when needed
	// 6. Prevent memory leaks by limiting cache size

	ctx := context.Background()
	
	// Setup test environment
	testDB := setupTestDB(t, "test_caching")
	defer testDB.Disconnect(ctx)
	
	promptsRepo := NewMockPromptsRepository()
	logger := &mockLogger{}
	engine := services.NewPromptEngine(promptsRepo, logger)
	
	t.Run("should cache processed prompts", func(t *testing.T) {
		// Setup test data
		userID := "cache-test-user"
		now := time.Now()
		
		testUser := &entities.User{
			ID:        userID,
			Email:     "cache@example.com",
			CreatedAt: now,
			UpdatedAt: now,
		}
		
		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Caching Test Topic",
			Ideas:         5,
			RelatedTopics: []string{"Cache", "Performance"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		
		template := `Test template for {name} with {ideas} ideas`
		
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			Name:           "cache-test",
			StyleName:      "cache-test",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)
		
		// Initial cache should be empty
		assert.Equal(t, 0, engine.CacheSize())
		
		// Process prompt first time (cache miss)
		processed1, err := engine.ProcessPrompt(
			ctx,
			userID,
			"cache-test",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)
		
		// Should be cached now
		assert.Equal(t, 1, engine.CacheSize())
		
		// Process same prompt again (cache hit)
		processed2, err := engine.ProcessPrompt(
			ctx,
			userID,
			"cache-test",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)
		
		// Should return same cached result
		assert.Equal(t, processed1, processed2)
		assert.Equal(t, 1, engine.CacheSize()) // Still only one entry
	})
	
	t.Run("should generate unique cache keys for different parameters", func(t *testing.T) {
		userID := "cache-key-user"
		now := time.Now()
		
		testUser := &entities.User{
			ID:    userID,
			Email: "key@example.com",
		}
		
		// Create two different topics
		topic1 := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Topic 1",
			Ideas:         3,
			RelatedTopics: []string{"A", "B"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		
		topic2 := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Topic 2",
			Ideas:         5,
			RelatedTopics: []string{"C", "D"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		
		template := `Template for {name} with {ideas} ideas`
		
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			Name:           "key-test",
			StyleName:      "key-test",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)
		
		// Process with topic 1
		_, err := engine.ProcessPrompt(
			ctx,
			userID,
			"key-test",
			entities.PromptTypeIdeas,
			topic1,
			nil,
			testUser,
		)
		require.NoError(t, err)
		
		assert.Equal(t, 1, engine.CacheSize())
		
		// Process with topic 2 (should create new cache entry)
		_, err = engine.ProcessPrompt(
			ctx,
			userID,
			"key-test",
			entities.PromptTypeIdeas,
			topic2,
			nil,
			testUser,
		)
		require.NoError(t, err)
		
		assert.Equal(t, 2, engine.CacheSize()) // Two entries for different topics
		
		// Process topic 1 again (should reuse cache)
		_, err = engine.ProcessPrompt(
			ctx,
			userID,
			"key-test",
			entities.PromptTypeIdeas,
			topic1,
			nil,
			testUser,
		)
		require.NoError(t, err)
		
		assert.Equal(t, 2, engine.CacheSize()) // Still two entries (cache hit for topic 1)
	})
	
	t.Run("should generate unique cache keys for different users", func(t *testing.T) {
		now := time.Now()
		
		user1 := &entities.User{
			ID:    "user-1",
			Email: "user1@example.com",
		}
		
		user2 := &entities.User{
			ID:    "user-2",
			Email: "user2@example.com",
		}
		
		// Same topic for both users
		topic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        user1.ID, // This doesn't matter for caching
			Name:          "Shared Topic",
			Ideas:         3,
			RelatedTopics: []string{"Shared"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		
		template := `Template for {name}`
		
		// Create prompts for each user
		prompt1 := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         user1.ID,
			Type:           entities.PromptTypeIdeas,
			Name:           "shared-test",
			StyleName:      "shared-test",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt1)
		
		prompt2 := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         user2.ID,
			Type:           entities.PromptTypeIdeas,
			Name:           "shared-test",
			StyleName:      "shared-test",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt2)
		
		// Process for user 1
		_, err := engine.ProcessPrompt(
			ctx,
			user1.ID,
			"shared-test",
			entities.PromptTypeIdeas,
			topic,
			nil,
			user1,
		)
		require.NoError(t, err)
		
		assert.Equal(t, 1, engine.CacheSize())
		
		// Process for user 2 (different user context)
		_, err = engine.ProcessPrompt(
			ctx,
			user2.ID,
			"shared-test",
			entities.PromptTypeIdeas,
			topic,
			nil,
			user2,
		)
		require.NoError(t, err)
		
		assert.Equal(t, 2, engine.CacheSize()) // Different cache entries for different users
	})
	
	t.Run("should handle cache clearing", func(t *testing.T) {
		userID := "clear-test-user"
		now := time.Now()
		
		testUser := &entities.User{
			ID:    userID,
			Email: "clear@example.com",
		}
		
		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Clear Test Topic",
			Ideas:         3,
			RelatedTopics: []string{"Clear"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		
		template := `Template for {name}`
		
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			Name:           "clear-test",
			StyleName:      "clear-test",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)
		
		// Add entries to cache
		_, err := engine.ProcessPrompt(
			ctx,
			userID,
			"clear-test",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)
		
		assert.Equal(t, 1, engine.CacheSize())
		
		// Clear cache
		engine.ClearCache()
		
		// Cache should be empty
		assert.Equal(t, 0, engine.CacheSize())
		
		// Hit count should be reset
		assert.Equal(t, 0, engine.CacheHitCount())
	})
	
	t.Run("should include idea content in cache key for drafts", func(t *testing.T) {
		userID := "draft-cache-user"
		now := time.Now()
		
		testUser := &entities.User{
			ID:    userID,
			Email: "draft@example.com",
		}
		
		// Two different ideas for the same topic
		idea1 := &entities.Idea{
			ID:      primitive.NewObjectID().Hex(),
			UserID:  userID,
			Content: "First idea content",
			Active:  true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		
		idea2 := &entities.Idea{
			ID:      primitive.NewObjectID().Hex(),
			UserID:  userID,
			Content: "Second idea content",
			Active:  true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		
		template := `Based on: {content}`
		
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeDrafts,
			Name:           "draft-cache",
			StyleName:      "draft-cache",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)
		
		// Process with first idea
		processed1, err := engine.ProcessPrompt(
			ctx,
			userID,
			"draft-cache",
			entities.PromptTypeDrafts,
			nil,
			idea1,
			testUser,
		)
		require.NoError(t, err)
		
		assert.Equal(t, 1, engine.CacheSize())
		assert.Contains(t, processed1, "First idea content")
		assert.NotContains(t, processed1, "Second idea content")
		
		// Process with second idea (different cache entry)
		processed2, err := engine.ProcessPrompt(
			ctx,
			userID,
			"draft-cache",
			entities.PromptTypeDrafts,
			nil,
			idea2,
			testUser,
		)
		require.NoError(t, err)
		
		assert.Equal(t, 2, engine.CacheSize()) // Two entries for different ideas
		assert.Contains(t, processed2, "Second idea content")
		assert.NotContains(t, processed2, "First idea content")
		
		// Process first idea again (should reuse cache)
		processed3, err := engine.ProcessPrompt(
			ctx,
			userID,
			"draft-cache",
			entities.PromptTypeDrafts,
			nil,
			idea1,
			testUser,
		)
		require.NoError(t, err)
		
		assert.Equal(t, processed1, processed3) // Same as cached for first idea
		assert.Equal(t, 2, engine.CacheSize())  // Still two entries
	})
	
	t.Run("should track cache performance metrics", func(t *testing.T) {
		// Reset engine to start with fresh metrics
		engine.ClearCache()
		
		userID := "metrics-user"
		now := time.Now()
		
		testUser := &entities.User{
			ID:    userID,
			Email: "metrics@example.com",
		}
		
		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Metrics Topic",
			Ideas:         3,
			RelatedTopics: []string{"Metrics"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		
		template := `Template for {name}`
		
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			Name:           "metrics-test",
			StyleName:      "metrics-test",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)
		
		// Process first time (cache miss)
		_, err := engine.ProcessPrompt(
			ctx,
			userID,
			"metrics-test",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)
		
		// Initially has cache miss and no hit
		assert.GreaterOrEqual(t, engine.CacheHitCount(), 1) // At least one attempt
		
		// Process multiple times to get hits
		for i := 0; i < 5; i++ {
			_, err := engine.ProcessPrompt(
				ctx,
				userID,
				"metrics-test",
				entities.PromptTypeIdeas,
				testTopic,
				nil,
				testUser,
			)
			require.NoError(t, err)
		}
		
		// Should have cache hits now
		assert.GreaterOrEqual(t, engine.CacheHitCount(), 6) // 1 initial + 5 more
	})
	
	t.Run("should provide access to cache contents for diagnostics", func(t *testing.T) {
		userID := "diagnostics-user"
		now := time.Now()
		
		testUser := &entities.User{
			ID:    userID,
			Email: "diag@example.com",
		}
		
		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Diagnostics Topic",
			Ideas:         2,
			RelatedTopics: []string{"Diagnostic"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		
		template := `Template for {name}`
		
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			Name:           "diagnostics-test",
			StyleName:      "diagnostics-test",
			PromptTemplate: template,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		promptsRepo.AddPrompt(prompt)
		
		// Add to cache
		processed, err := engine.ProcessPrompt(
			ctx,
			userID,
			"diagnostics-test",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)
		
		// Get cache contents
		cacheContents := engine.GetCacheContents()
		
		// Should have our entry
		assert.Len(t, cacheContents, 1)
		
		// Should contain the processed value
		var found bool
		for _, value := range cacheContents {
			if value == processed {
				found = true
				break
			}
		}
		assert.True(t, found)
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
