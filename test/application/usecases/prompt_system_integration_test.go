package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/services"
)

// NOTE: These tests are written according to TDD Red pattern - they will FAIL
// until the prompt system integration is implemented

func TestPromptSystemIntegration(t *testing.T) {
	// Test for integration of the prompt system in the generation flow:
	// - GenerateIdeasUseCase uses PromptEngine for prompt processing
	// - GenerateDraftsUseCase uses PromptEngine for dynamic prompts
	// - System logs prompt usage and errors
	// - Fallback system works when prompts are missing
	// - End-to-end flow with all components working together

	// Setup test context
	ctx := context.Background()
	now := time.Now()

	// Setup test user
	userID := "test-integration-user-123"
	testUser := &entities.User{
		ID:        userID,
		Email:     "integration@test.com",
		Industry:  "Software Development",
		Role:      "Tech Lead",
		Experience: "3 years",
		Goals:     "Team leadership and system architecture",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Setup test topic
	testTopic := &entities.Topic{
		ID:              "topic-1",
		UserID:          userID,
		Name:            "Microservices Architecture Best Practices",
		Ideas:           5,
		RelatedTopics:   []string{"System Design", "DevOps", "Scalability"},
		Active:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Setup test idea
	testIdea := &entities.Idea{
		ID:        "idea-1",
		UserID:    userID,
		TopicID:   testTopic.ID,
		Content:   "Implementing service meshes in microservices for better observability and communication",
		Selected:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Mock repositories and services
	mockUserRepo := &MockUserRepository{users: []*entities.User{testUser}}
	mockTopicRepo := &MockTopicRepository{topics: []*entities.Topic{testTopic}}
	mockIdeasRepo := &MockIdeasRepository{ideas: []*entities.Idea{testIdea}}
	mockPromptsRepo := &MockPromptsRepository{}
	mockLLMService := &MockLLMService{}

	// Setup prompt engine
	promptEngine := services.NewPromptEngine(mockPromptsRepo)

	// Setup prompts
	ideasPrompt := &entities.Prompt{
		ID:             "ideas-prompt-1",
		UserID:         userID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Genera {ideas} ideas únicas sobre {name} considerando {[related_topics]}",
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	draftsPrompt := &entities.Prompt{
		ID:             "drafts-prompt-1",
		UserID:         userID,
		Name:           "professional",
		Type:           entities.PromptTypeDrafts,
		PromptTemplate: "Escribe un post profesional sobre {content} con el contexto: {user_context}",
		StyleName:      "professional",
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	setup := func() {
		mockPromptsRepo.prompts = []*entities.Prompt{ideasPrompt, draftsPrompt}
		// Clear any previous mock data
		mockLLMService.generatedIdeas = nil
		mockLLMService.generatedDrafts = nil
		mockIdeasRepo.savedIdeas = []*entities.Idea{}
	}

	t.Run("should use PromptEngine in GenerateIdeasUseCase", func(t *testing.T) {
		setup()

		// GIVEN a GenerateIdeasUseCase with PromptEngine integration
		useCase := NewGenerateIdeasUseCaseWithPromptEngine(
			mockUserRepo,
			mockTopicRepo,
			mockIdeasRepo,
			mockPromptsRepo,
			mockLLMService,
			promptEngine,
		)

		// WHEN generating ideas
		ideas, err := useCase.GenerateIdeasForUser(ctx, userID, 5)

		// THEN the PromptEngine should be used to process the prompt
		require.NoError(t, err)
		assert.Len(t, ideas, 5)

		// Verify the prompt was processed with variables
		assert.Contains(t, mockLLMService.lastPromptUsed, "Genera 5 ideas únicas")
		assert.Contains(t, mockLLMService.lastPromptUsed, "Microservices Architecture Best Practices")
		assert.Contains(t, mockLLMService.lastPromptUsed, "System Design, DevOps, Scalability")

		// Verify ideas were created and saved
		assert.Len(t, mockIdeasRepo.savedIdeas, 5)
		for _, idea := range mockIdeasRepo.savedIdeas {
			assert.Equal(t, userID, idea.UserID)
			assert.Equal(t, testTopic.ID, idea.TopicID)
			assert.NotEmpty(t, idea.Content)
		}
	})

	t.Run("should use PromptEngine in GenerateDraftsUseCase", func(t *testing.T) {
		setup()

		// GIVEN a GenerateDraftsUseCase with PromptEngine integration
		useCase := NewGenerateDraftsUseCaseWithPromptEngine(
			mockUserRepo,
			mockIdeasRepo,
			mockDraftRepo,
			mockLLMService,
			promptEngine,
			mockPromptsRepo,
		)

		// WHEN generating drafts
		drafts, err := useCase.Execute(ctx, GenerateDraftsInput{
			UserID: userID,
			IdeaID: testIdea.ID,
		})

		// THEN the PromptEngine should be used to process the prompt
		require.NoError(t, err)
		assert.Len(t, drafts, 6) // 5 posts + 1 article

		// Verify the prompt was processed with content and user context
		assert.Contains(t, mockLLMService.lastPromptUsed, "Escribe un post profesional")
		assert.Contains(t, mockLLMService.lastPromptUsed, "Implementing service meshes")
		assert.Contains(t, mockLLMService.lastPromptUsed, "integration@test.com")
		assert.Contains(t, mockLLMService.lastPromptUsed, "Tech Lead")

		// Verify drafts were created
		assert.Len(t, mockDraftRepo.savedDrafts, 6)
	})

	t.Run("should fall back to default prompts when user has no prompts", func(t *testing.T) {
		// GIVEN a user without custom prompts
		userWithoutPrompts := "user-no-prompts"
		userWithoutPromptsEntity := &entities.User{
			ID:        userWithoutPrompts,
			Email:     "noprompts@test.com",
			Industry:  "Education",
			Role:      "Teacher",
			Experience: "2 years",
			Goals:     "Student engagement techniques",
			CreatedAt: now,
			UpdatedAt: now,
		}

		mockUserRepo.users = append(mockUserRepo.users, userWithoutPromptsEntity)
		mockPromptsRepo.prompts = []*entities.Prompt{} // No custom prompts

		// Create topic for this user
		topicWithoutPrompts := &entities.Topic{
			ID:              "topic-2",
			UserID:          userWithoutPrompts,
			Name:            "Digital Learning Tools",
			Ideas:           3,
			RelatedTopics:   []string{"Technology", "Education"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		mockTopicRepo.topics = append(mockTopicRepo.topics, topicWithoutPrompts)

		// GIVEN a GenerateIdeasUseCase
		useCase := NewGenerateIdeasUseCaseWithPromptEngine(
			mockUserRepo,
			mockTopicRepo,
			mockIdeasRepo,
			mockPromptsRepo,
			mockLLMService,
			promptEngine,
		)

		// WHEN generating ideas
		ideas, err := useCase.GenerateIdeasForUser(ctx, userWithoutPrompts, 3)

		// THEN should fall back to default prompt and still generate ideas
		require.NoError(t, err)
		assert.Len(t, ideas, 3)

		// Verify the default prompt was used
		assert.Contains(t, mockLLMService.lastPromptUsed, "Genera 3 ideas de contenido")
		assert.Contains(t, mockLLMService.lastPromptUsed, "Digital Learning Tools")
	})

	t.Run("should log prompt system usage and errors", func(t *testing.T) {
		setup()

		// GIVEN a GenerateIdeasUseCase with logging
		useCase := NewGenerateIdeasUseCaseWithPromptEngine(
			mockUserRepo,
			mockTopicRepo,
			mockIdeasRepo,
			mockPromptsRepo,
			mockLLMService,
			promptEngine,
		)

		// WHEN generating ideas
		ideas, err := useCase.GenerateIdeasForUser(ctx, userID, 5)
		require.NoError(t, err)

		// THEN prompt system activity should be logged
		logEntries := promptEngine.GetLogEntries()
		assert.Greater(t, len(logEntries), 0)

		// Find log entry for this user's prompt usage
		var userLogEntry *services.PromptLogEntry
		for _, entry := range logEntries {
			if entry.UserID == userID && entry.Action == "process" {
				userLogEntry = entry
				break
			}
		}

		require.NotNil(t, userLogEntry)
		assert.Equal(t, "base1", userLogEntry.PromptName)
		assert.Equal(t, string(entities.PromptTypeIdeas), userLogEntry.PromptType)
		assert.True(t, userLogEntry.Success)
	})

	t.Run("should handle prompt errors gracefully", func(t *testing.T) {
		setup()

		// GIVEN a prompt with invalid template syntax
		invalidPrompt := &entities.Prompt{
			ID:             "invalid-prompt",
			UserID:         userID,
			Name:           "invalid",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Template with {invalid_variable syntax",
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		mockPromptsRepo.prompts = append(mockPromptsRepo.prompts, invalidPrompt)

		// AND a GenerateIdeasUseCase
		useCase := NewGenerateIdeasUseCaseWithPromptEngine(
			mockUserRepo,
			mockTopicRepo,
			mockIdeasRepo,
			mockPromptsRepo,
			mockLLMService,
			promptEngine,
		)

		// WHEN generating ideas
		ideas, err := useCase.GenerateIdeasForUser(ctx, userID, 5)

		// THEN should handle the error gracefully
		require.Error(t, err)
		assert.Nil(t, ideas)
		assert.Contains(t, err.Error(), "invalid template")

		// AND should log the error
		logEntries := promptEngine.GetLogEntries()
		
		// Find error log entry
		var errorLogEntry *services.PromptLogEntry
		for _, entry := range logEntries {
			if entry.UserID == userID && entry.Action == "error" {
				errorLogEntry = entry
				break
			}
		}

		require.NotNil(t, errorLogEntry)
		assert.Equal(t, "invalid", errorLogEntry.PromptName)
		assert.False(t, errorLogEntry.Success)
		assert.Contains(t, errorLogEntry.ErrorMessage, "invalid template")
	})

	t.Run("should use cached prompts when available", func(t *testing.T) {
		setup()

		// GIVEN a GenerateIdeasUseCase
		useCase := NewGenerateIdeasUseCaseWithPromptEngine(
			mockUserRepo,
			mockTopicRepo,
			mockIdeasRepo,
			mockPromptsRepo,
			mockLLMService,
			promptEngine,
		)

		// WHEN generating ideas for the same topic twice
		ideas1, err := useCase.GenerateIdeasForUser(ctx, userID, 5)
		require.NoError(t, err)
		assert.Len(t, ideas1, 5)

		// Create new idea instance to simulate second request with same topic
		mockIdeasRepo.savedIdeas = []*entities.Idea{}
		ideas2, err := useCase.GenerateIdeasForUser(ctx, userID, 5)
		require.NoError(t, err)
		assert.Len(t, ideas2, 5)

		// THEN the prompt should be cached
		assert.Equal(t, mockLLMService.lastPromptUsed, mockLLMService.cachedPromptUsed)
		assert.True(t, promptEngine.GetCacheHitCount() > 0)
	})

	t.Run("should provide diagnostic endpoints for prompt system", func(t *testing.T) {
		setup()

		// WHEN requesting prompt system diagnostics
		diagnostics := promptEngine.GetDiagnostics(ctx, userID)

		// THEN should include comprehensive system information
		require.NotNil(t, diagnostics)
		
		assert.True(t, diagnostics.PromptEngineActive)
		assert.Equal(t, 2, diagnostics.UserPromptCount)
		assert.GreaterOrEqual(t, diagnostics.CacheSize, 0)
		assert.Contains(t, diagnostics.SupportedVariables, "name")
		assert.Contains(t, diagnostics.SupportedVariables, "ideas")
		assert.Contains(t, diagnostics.SupportedVariables, "content")
		assert.Contains(t, diagnostics.SupportedVariables, "user_context")
		assert.Contains(t, diagnostics.SupportedVariables, "[related_topics]")
	})
}

// NEW types that don't exist yet (to be implemented)

type GenerateIdeasUseCaseWithPromptEngine struct {
	// TODO: Implementation needed - this test will fail until implemented
}

type GenerateDraftsUseCaseWithPromptEngine struct {
	// TODO: Implementation needed - this test will fail until implemented
}

// New constructors to be implemented
func NewGenerateIdeasUseCaseWithPromptEngine(
	userRepo *MockUserRepository,
	topicRepo *MockTopicRepository,
	ideasRepo *MockIdeasRepository,
	promptsRepo *MockPromptsRepository,
	llmService *MockLLMService,
	promptEngine *services.PromptEngine,
) *GenerateIdeasUseCaseWithPromptEngine {
	// TODO: Implementation needed - this test will fail until implemented
	return nil
}

func NewGenerateDraftsUseCaseWithPromptEngine(
	userRepo *MockUserRepository,
	ideasRepo *MockIdeasRepository,
	draftRepo *MockDraftRepository,
	llmService *MockLLMService,
	promptEngine *services.PromptEngine,
	promptsRepo *MockPromptsRepository,
) *GenerateDraftsUseCaseWithPromptEngine {
	// TODO: Implementation needed - this test will fail until implemented
	return nil
}

// Mock implementations needed for the test
type MockDraftRepository struct {
	savedDrafts []*entities.Draft
}

func (m *MockDraftRepository) Create(ctx context.Context, draft *entities.Draft) error {
	m.savedDrafts = append(m.savedDrafts, draft)
	return nil
}

func (m *MockDraftRepository) FindByIdeaID(ctx context.Context, ideaID string) ([]*entities.Draft, error) {
	// Not needed for this test
	return nil, assert.AnError
}

func (m *MockDraftRepository) FindByID(ctx context.Context, id string) (*entities.Draft, error) {
	// Not needed for this test
	return nil, assert.AnError
}

func (m *MockDraftRepository) Delete(ctx context.Context, id string) error {
	// Not needed for this test
	return assert.AnError
}

// Extended MockIdeasRepository to track saved ideas
func (m *MockIdeasRepository) Create(ctx context.Context, idea *entities.Idea) error {
	m.savedIdeas = append(m.savedIdeas, idea)
	return nil
}

// Additional PromptEngine methods to be implemented
func (p *services.PromptEngine) GetLogEntries() []*services.PromptLogEntry {
	// TODO: Implementation needed - this test will fail until implemented
	return nil
}

func (p *services.PromptEngine) GetCacheHitCount() int {
	// TODO: Implementation needed - this test will fail until implemented
	return 0
}

func (p *services.PromptEngine) GetDiagnostics(ctx context.Context, userID string) *services.PromptDiagnostics {
	// TODO: Implementation needed - this test will fail until implemented
	return nil
}

// Types to be implemented in the services package
type PromptLogEntry struct {
	UserID      string
	PromptName  string
	PromptType  string
	Action      string
	Timestamp   time.Time
	Success     bool
	ErrorMessage string
}

type PromptDiagnostics struct {
	PromptEngineActive  bool
	UserPromptCount     int
	CacheSize          int
	SupportedVariables []string
}
