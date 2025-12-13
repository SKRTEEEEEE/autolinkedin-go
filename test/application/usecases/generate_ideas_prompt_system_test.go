package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/linkgen-ai/backend/src/application/usecases"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/test/mocks"
)

// TestGenerateIdeasWithPromptSystem tests the generate ideas use case with new prompt system
// REQUIRES: LLM at http://100.105.212.98:8317/ (not mocked)
func TestGenerateIdeasWithPromptSystem(tt *testing.T) {
	if testing.Short() {
		tt.Skip("Skipping integration test in short mode - requires LLM endpoint")
	}

	ctx := context.Background()

	tt.Run("should generate ideas using custom prompt with variable replacement", func(t *testing.T) {
		// GIVEN a custom prompt with multiple variables
		userID := primitive.NewObjectID().Hex()
		topicID := primitive.NewObjectID().Hex()
		
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "custom-ideas",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Generate {ideas} innovative ideas about {name} focusing on {[keywords]} with priority level {priority}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		topic := &entities.Topic{
			ID:            topicID,
			UserID:        userID,
			Name:          "Microservices Architecture",
			Description:   "Building scalable distributed systems",
			Keywords:      []string{"docker", "kubernetes", "api-gateway", "service-discovery"},
			Category:      "Backend",
			Priority:      9,
			IdeasCount:    5,
			Active:        true,
			PromptName:    "custom-ideas",
			RelatedTopics: []string{"Serverless", "Event-Driven Architecture"},
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		// Mock repositories - will need actual implementation
		promptRepo := &mocks.MockPromptRepository{}
		topicRepo := &mocks.MockTopicRepository{}
		ideaRepo := &mocks.MockIdeaRepository{}
		llmClient := &mocks.MockLLMClient{}

		// Setup expectations
		promptRepo.On("GetActiveByName", ctx, userID, "custom-ideas").Return(prompt, nil)
		topicRepo.On("GetByID", ctx, topicID).Return(topic, nil)
		
		// Mock LLM response - should NOT be mocked, should call real endpoint
		// llmClient will be replaced with actual HTTP client to http://100.105.212.98:8317/
		
		// WHEN generating ideas with the custom prompt
		useCase := usecases.NewGenerateIdeasUseCase(promptRepo, topicRepo, ideaRepo, llmClient)
		
		req := &usecases.GenerateIdeasRequest{
			UserID:  userID,
			TopicID: topicID,
			Count:   5,
		}

		// This will fail until the prompt system is properly implemented
		t.Fatal("implement generate ideas with prompt system - FAILING IN TDD RED PHASE")
	})

	tt.Run("should fallback to default prompt when topic has no prompt reference", func(t *testing.T) {
		// GIVEN a topic without prompt reference
		userID := primitive.NewObjectID().Hex()
		topicID := primitive.NewObjectID().Hex()

		topic := &entities.Topic{
			ID:            topicID,
			UserID:        userID,
			Name:          "React Hooks",
			Description:   "Modern React patterns",
			Keywords:      []string{"react", "hooks", "functional-components"},
			Category:      "Frontend",
			Priority:      7,
			IdeasCount:    3,
			Active:        true,
			PromptName:    "", // Empty - should fallback
			RelatedTopics: []string{"JavaScript ES6", "State Management"},
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		defaultPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Generate {ideas} ideas about {name} with {[related_topics]}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		promptRepo := &mocks.MockPromptRepository{}
		topicRepo := &mocks.MockTopicRepository{}
		ideaRepo := &mocks.MockIdeaRepository{}
		llmClient := &mocks.MockLLMClient{}

		promptRepo.On("GetDefault", ctx, userID).Return(defaultPrompt, nil)
		topicRepo.On("GetByID", ctx, topicID).Return(topic, nil)

		// WHEN generating ideas without specific prompt
		useCase := usecases.NewGenerateIdeasUseCase(promptRepo, topicRepo, ideaRepo, llmClient)
		
		req := &usecases.GenerateIdeasRequest{
			UserID:  userID,
			TopicID: topicID,
			Count:   3,
		}

		// This will fail until fallback logic is implemented
		t.Fatal("implement prompt fallback logic - FAILING IN TDD RED PHASE")
	})

	tt.Run("should validate idea content length according to entity.md specifications", func(t *testing.T) {
		// GIVEN a generation request
		userID := primitive.NewObjectID().Hex()
		topicID := primitive.NewObjectID().Hex()

		// Mock setup
		promptRepo := &mocks.MockPromptRepository{}
		topicRepo := &mocks.MockTopicRepository{}
		ideaRepo := &mocks.MockIdeaRepository{}
		llmClient := &mocks.MockLLMClient{}

		// WHEN generating ideas
		useCase := usecases.NewGenerateIdeasUseCase(promptRepo, topicRepo, ideaRepo, llmClient)
		
		req := &usecases.GenerateIdeasRequest{
			UserID:  userID,
			TopicID: topicID,
			Count:   5,
		}

		// This will fail until validation is implemented
		t.Fatal("implement idea content length validation (10-200 chars) - FAILING IN TDD RED PHASE")
		
		// THEN should validate each idea:
		// - Minimum length: 10 characters
		// - Maximum length: 200 characters (as defined in entity.md)
		// Should reject ideas outside these bounds
	})
}

// TestGenerateDraftsWithPromptSystem tests draft generation with user context
func TestGenerateDraftsWithPromptSystem(tt *testing.T) {
	if testing.Short() {
		tt.Skip("Skipping integration test in short mode - requires LLM endpoint")
	}

	ctx := context.Background()

	tt.Run("should generate drafts using prompt with user context variables", func(t *testing.T) {
		// GIVEN an idea and user context
		userID := primitive.NewObjectID().Hex()
		ideaID := primitive.NewObjectID().Hex()
		topicID := primitive.NewObjectID().Hex()

		draftPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "professional-draft",
			Type:           entities.PromptTypeDrafts,
			PromptTemplate: `Create professional LinkedIn content based on:
Idea: {content}
User Context: {user_context}
Topic: {topic_name}

Generate 5 posts and 1 article in Spanish.`,
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		idea := &entities.Idea{
			ID:        ideaID,
			Content:   "Implementando CQRS en microservicios con Go",
			TopicID:   topicID,
			TopicName: "Go Microservices",
			UserID:    userID,
		}

		user := &entities.User{
			ID:   userID,
			Name: "Juan García",
			Configuration: map[string]interface{}{
				"name":             "Juan García",
				"expertise":        "Desarrollo Backend",
				"tone_preference":  "Profesional",
			},
		}

		// Mock repositories
		promptRepo := &mocks.MockPromptRepository{}
		ideaRepo := &mocks.MockIdeaRepository{}
		userRepo := &mocks.MockUserRepository{}
		llmClient := &mocks.MockLLMClient{}

		promptRepo.On("GetActiveByType", ctx, userID, entities.PromptTypeDrafts).Return(draftPrompt, nil)
		ideaRepo.On("GetByID", ctx, ideaID).Return(idea, nil)
		userRepo.On("GetByID", ctx, userID).Return(user, nil)

		// WHEN generating drafts with user context
		useCase := usecases.NewGenerateDraftsUseCase(promptRepo, ideaRepo, userRepo, llmClient)
		
		req := &usecases.GenerateDraftsRequest{
			UserID: userID,
			IdeaID: ideaID,
		}

		// This will fail until draft generation with user context is implemented
		t.Fatal("implement draft generation with user context variables - FAILING IN TDD RED PHASE")
		
		// THEN should generate:
		// - 5 LinkedIn posts
		// - 1 article
		// With proper variable replacement:
		// - {content} -> idea content
		// - {user_context} -> formatted user context
		// - {topic_name} -> topic name
	})

	tt.Run("should format user context according to seed/README.md specifications", func(t *testing.T) {
		// GIVEN a user with configuration
		user := &entities.User{
			ID:   primitive.NewObjectID().Hex(),
			Name: "María Rodríguez",
			Configuration: map[string]interface{}{
				"name":            "María Rodríguez",
				"expertise":       "Full Stack Development",
				"tone_preference": "Educativo",
				"industry":        "Tecnología",
			},
		}

		// WHEN building user context string
		// This will fail until user context formatting is implemented
		t.Fatal("implement user context formatting per seed/README.md - FAILING IN TDD RED PHASE")
		
		// THEN should format as:
		// Name: María Rodríguez
		// Expertise: Full Stack Development
		// Tone: Educativo
		// (Industry should be ignored if not in specification)
	})

	tt.Run("should handle missing user configuration gracefully", func(t *testing.T) {
		// GIVEN a user with minimal configuration
		user := &entities.User{
			ID:       primitive.NewObjectID().Hex(),
			Name:     "Test User",
			Configuration: map[string]interface{}{
				// Missing name, expertise, and tone_preference
			},
		}

		// WHEN building user context with missing fields
		// This will fail until missing field handling is implemented
		t.Fatal("implement handling of missing user configuration fields - FAILING IN TDD RED PHASE")
		
		// THEN should use empty strings or defaults for missing fields
	})
}
