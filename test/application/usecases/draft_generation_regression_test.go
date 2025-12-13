package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/test/mocks"
)

// TestDraftGenerationRegression tests that draft generation still produces expected outputs
// after the prompt system refactor
func TestDraftGenerationRegression(tt *testing.T) {
	if testing.Short() {
		tt.Skip("Skipping regression test in short mode - requires LLM endpoint")
	}

	ctx := context.Background()

	tt.Run("should generate same number of drafts as before refactor", func(t *testing.T) {
		// GIVEN pre-refactor expected output structure
		userID := primitive.NewObjectID().Hex()
		idea := &entities.Idea{
			ID:        primitive.NewObjectID().Hex(),
			Content:   "Implementación de microservicios con Go y gRPC",
			TopicName: "Go Backend Development",
			UserID:    userID,
		}

		user := &entities.User{
			ID:   primitive.NewObjectID(),
			Name: "Juan García",
			Configuration: map[string]interface{}{
				"name":            "Juan García",
				"expertise":       "Backend Development",
				"tone_preference":  "Profesional",
			},
		}

		// Setup mock repositories
		promptRepo := &mocks.MockPromptRepository{}
		ideaRepo := &mocks.MockIdeaRepository{}
		userRepo := &mocks.MockUserRepository{}
		llmClient := &mocks.MockLLMClient{}

		userRepo.On("GetByID", ctx, userID).Return(user, nil)
		ideaRepo.On("GetByID", ctx, idea.ID).Return(idea, nil)
		promptRepo.On("GetActiveByType", ctx, userID, entities.PromptTypeDrafts).Return(
			&entities.Prompt{
				Name:           "professional",
				Type:           entities.PromptTypeDrafts,
				PromptTemplate: "Generate content based on: {content}\nUser: {user_context}",
				Active:         true,
			},
			nil,
		)

		// WHEN generating drafts with new prompt system
		useCase := usecases.NewGenerateDraftsUseCase(promptRepo, ideaRepo, userRepo, llmClient)
		
		req := &usecases.GenerateDraftsRequest{
			UserID: userID,
			IdeaID: idea.ID,
		}

		// This will fail until draft generation is updated with new system
		t.Fatal("implement draft generation regression with new prompt system - FAILING IN TDD RED PHASE")

		response, err := useCase.Execute(ctx, req)
		require.NoError(t, err)

		// THEN should maintain same output structure
		assert.Equal(t, 5, len(response.Posts), "Should generate exactly 5 posts")
		assert.Equal(t, 1, len(response.Articles), "Should generate exactly 1 article")
		
		// Verify post characteristics
		for i, post := range response.Posts {
			assert.NotEmpty(t, post.Content, fmt.Sprintf("Post %d should have content", i+1))
			
			// Posts should be reasonable length for LinkedIn
			assert.Greater(t, len(post.Content), 50, "Post should have meaningful content")
			assert.Less(t, len(post.Content), 300, "Post should not exceed Twitter-like limit")
			
			// Should include Spanish content as expected
			assert.True(t, containsSpanishWords(post.Content), "Post should be in Spanish")
		}

		// Verify article characteristics
		article := response.Articles[0]
		assert.NotEmpty(t, article.Title, "Article should have title")
		assert.NotEmpty(t, article.Content, "Article should have content")
		assert.Greater(t, len(article.Content), 500, "Article should be substantial")
		assert.True(t, containsSpanishWords(article.Content), "Article should be in Spanish")
	})

	tt.Run("should maintain JSON response format for LLM communication", func(t *testing.T) {
		// GIVEN pre-refactor expected LLM communication format
		userID := primitive.NewObjectID().Hex()
		idea := &entities.Idea{
			Content:   "Optimización de consultas SQL en aplicaciones Node.js",
			TopicName: "Database Optimization",
			UserID:    userID,
		}

		user := &entities.User{
			Configuration: map[string]interface{}{
				"name":            "María López",
				"expertise":       "Full Stack",
				"tone_preference":  "Técnico",
			},
		}

		llmClient := &mocks.MockLLMClient{}
		
		// WHEN generating drafts
		useCase := setupDraftUseCaseWithClients(llmClient)
		
		req := &usecases.GenerateDraftsRequest{
			UserID: userID,
			IdeaID: idea.ID,
		}

		// This will fail until LLM communication format is maintained
		t.Fatal("implement backward-compatible LLM JSON communication - FAILING IN TDD RED PHASE")

		_, err := useCase.Execute(ctx, req)
		require.NoError(t, err)

		// THEN should maintain same JSON structure sent to LLM
		llmCall := llmClient.GetLastCall()
		require.NotNil(t, llmCall, "Should have made LLM call")
		require.NotEmpty(t, llmCall.Prompt, "Should have sent prompt to LLM")

		// Verify prompt contains expected JSON structure
		assert.Contains(t, llmCall.Prompt, `"posts"`, "Prompt should request posts array")
		assert.Contains(t, llmCall.Prompt, `"articles"`, "Prompt should request articles array")

		// Verify prompt contains the idea content
		assert.Contains(t, llmCall.Prompt, idea.Content, "Should include original idea")
		
		// Verify user context is properly included
		assert.Contains(t, llmCall.Prompt, "María López", "Should include user name")
		assert.Contains(t, llmCall.Prompt, "Full Stack", "Should include user expertise")

		// Verify variable substitution worked
		assert.Contains(t, llmCall.Prompt, "{user_context}", "Should have replaced user context variable")
	})

	tt.Run("should handle malformed LLM responses gracefully", func(t *testing.T) {
		// GIVEN LLM client that returns malformed responses
		malResponses := []string{
			"Broken JSON {posts: [",
			`{"posts": ["Only one post", ""]}`, // Too few posts
			`{"posts": ["post1", "post2", "post3", "post4", "post5", "too many"]}`, // Too many posts
			`{"posts": ["Very long post that exceeds the reasonable length limit for a LinkedIn post and should be rejected during validation because it goes well beyond what would be considered appropriate for the platform"]}`,
			`{"posts": []}`, // Empty posts
		}

		for i, malformedResponse := range malResponses {
			t.Run(fmt.Sprintf("malformed scenario %d", i+1), func(t *testing.T) {
				llmClient := &mocks.MockLLMClient{}
				llmClient.SetFixedResponse(malformedResponse)

				useCase := setupDraftUseCaseWithClients(llmClient)
				
				req := &usecases.GenerateDraftsRequest{
					UserID: primitive.NewObjectID().Hex(),
					IdeaID: primitive.NewObjectID().Hex(),
				}

				// WHEN handling malformed response
				response, err := useCase.Execute(ctx, req)

				// THEN should handle gracefully
				// This will fail until error handling is implemented
				t.Fatal("implement graceful handling of malformed LLM responses - FAILING IN TDD RED PHASE")

				// Either should return error with helpful message
				// Or should attempt to recover/process partial response
				if err != nil {
					assert.Contains(t, err.Error(), "LLM response", "Error should indicate LLM issue")
				} else {
					// If not error, should have some fallback behavior
					assert.NotNil(t, response, "Should have response even with malformed LLM output")
				}
			})
		}
	})

	tt.Run("should maintain performance characteristics for draft generation", func(t *testing.T) {
		// GIVEN draft generation requirements
		numDrafts := 20 // Generate multiple drafts to test performance
		
		// Track timing
		var durations []time.Duration
		successCount := 0
		errorCount := 0

		for i := 0; i < numDrafts; i++ {
			userID := primitive.NewObjectID().Hex()
			
			req := &usecases.GenerateDraftsRequest{
				UserID: userID,
				IdeaID: primitive.NewObjectID().Hex(),
			}

			useCase := setupDraftUseCaseWithClients(&mocks.MockLLMClient{})

			// WHEN measuring performance
			start := time.Now()
			
			// This will fail until performance is maintained
			t.Fatal("implement maintaining draft generation performance after refactor - FAILING IN TDD RED PHASE")

			_, err := useCase.Execute(ctx, req)
			duration := time.Since(start)
			
			durations = append(durations, duration)
			
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
		}

		// THEN should maintain performance
		if successCount > 0 {
			var totalDuration time.Duration
			for _, d := range durations {
				totalDuration += d
			}
			avgDuration := totalDuration / time.Duration(len(durations))
			
			t.Logf("Draft generation performance: %d successful, %d errors", successCount, errorCount)
			t.Logf("Average time: %v", avgDuration)
			
			// Performance assertions
			assert.Greater(t, successCount, numDrafts*8/10, "At least 80% should succeed")
			assert.Less(t, avgDuration, 15*time.Second, "Average should be under 15 seconds")
			assert.Less(t, totalDuration, 300*time.Second, "Total should complete within 5 minutes")
		}
	})
}

// TestPromptIntegrationRegression tests integration between prompts and draft generation
func TestPromptIntegrationRegression(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should use draft prompts correctly for idea generation", func(t *testing.T) {
		// GIVEN different types of draft prompts
		userID := primitive.NewObjectID().Hex()
		
		prompts := []*entities.Prompt{
			{
				ID:             primitive.NewObjectID().Hex(),
				Name:           "professional",
				Type:           entities.PromptTypeDrafts,
				PromptTemplate: "Create professional content about {topic_name} based on: {content}",
				Active:         true,
			},
			{
				ID:             primitive.NewObjectID().Hex(),
				Name:           "casual",
				Type:           entities.PromptTypeDrafts,
				PromptTemplate: "Write casual posts about {topic_name} from idea: {content}",
				Active:         true,
			},
			{
				ID:             primitive.NewObjectID().Hex(),
				Name:           "technical",
				Type:           entities.PromptTypeDrafts,
				PromptTemplate: "Generate technical deep-dive content for {topic_name}: {content}",
				Active:         true,
			},
		}

		idea := &entities.Idea{
			Content:   "Kubernetes best practices for production deployments",
			TopicName: "DevOps",
			UserID:    userID,
		}

		for i, prompt := range prompts {
			t.Run(fmt.Sprintf("using %s prompt", prompt.Name), func(t *testing.T) {
				// Setup mocks
				promptRepo := &mocks.MockPromptRepository{}
				ideaRepo := &mocks.MockIdeaRepository{}
				userRepo := &mocks.MockUserRepository{}
				llmClient := &mocks.MockLLMClient{}

				promptRepo.On("GetActiveByType", ctx, userID, entities.PromptTypeDrafts).Return(prompt, nil)
				ideaRepo.On("GetByID", ctx, idea.ID).Return(idea, nil)
				userRepo.On("GetByID", ctx, userID).Return(&entities.User{}, nil)

				useCase := usecases.NewGenerateDraftsUseCase(promptRepo, ideaRepo, userRepo, llmClient)
				
				req := &usecases.GenerateDraftsRequest{
					UserID: userID,
					IdeaID: idea.ID,
				}

				// WHEN generating with specific prompt
				// This will fail until prompt usage is properly implemented
				t.Fatal("implement proper draft prompt usage - FAILING IN TDD RED PHASE")

				response, err := useCase.Execute(ctx, req)
				require.NoError(t, err)

				// THEN should use prompt correctly
				assert.Equal(t, 5, len(response.Posts), "Should always generate 5 posts regardless of prompt")
				assert.Equal(t, 1, len(response.Articles), "Should always generate 1 article")

				// Verify prompt was actually used
				llmCall := llmClient.GetLastCall()
				require.NotNil(t, llmCall)
				
				assert.Contains(t, llmCall.Prompt, prompt.PromptTemplate, 
					"Should use specific prompt template")
				assert.Contains(t, llmCall.Prompt, idea.Content, 
					"Should include idea in prompt")
				assert.Contains(t, llmCall.Prompt, idea.TopicName, 
					"Should include topic name variable")

				// Content should reflect prompt style
				for j, post := range response.Posts {
					assert.NotEmpty(t, post.Content, 
						fmt.Sprintf("Post %d with %s prompt should have content", j+1, prompt.Name))
					
					// Content should vary based on prompt type
					switch prompt.Name {
					case "professional":
						assert.True(t, isProfessionalTone(post.Content), 
							"Should have professional tone")
					case "casual":
						assert.True(t, isCasualTone(post.Content), 
							"Should have casual tone")
					case "technical":
						assert.True(t, isTechnicalTone(post.Content), 
							"Should have technical depth")
					}
				}
			})
		}
	})

	tt.Run("should fallback to default draft prompt when no specific prompt is active", func(t *testing.T) {
		// GIVEN no active draft prompt
		userID := primitive.NewObjectID().Hex()
		
		promptRepo := &mocks.MockPromptRepository{}
		ideaRepo := &mocks.MockIdeaRepository{}
		userRepo := &mocks.MockUserRepository{}
		llmClient := &mocks.MockLLMClient{}

		// No active draft prompts
		promptRepo.On("GetActiveByType", ctx, userID, entities.PromptTypeDrafts).Return(nil, nil)
		
		idea := &entities.Idea{
			Content:   "Machine learning models for time series prediction",
			TopicName: "AI/ML",
			UserID:    userID,
		}
		
		ideaRepo.On("GetByID", ctx, idea.ID).Return(idea, nil)
		userRepo.On("GetByID", ctx, userID).Return(&entities.User{}, nil)

		useCase := usecases.NewGenerateDraftsUseCase(promptRepo, ideaRepo, userRepo, llmClient)
		
		req := &usecases.GenerateDraftsRequest{
			UserID: userID,
			IdeaID: idea.ID,
		}

		// WHEN generating without specific prompt
		// This will fail until fallback logic is implemented
		t.Fatal("implement fallback to default draft prompt - FAILING IN TDD RED PHASE")

		response, err := useCase.Execute(ctx, req)
		require.NoError(t, err)

		// THEN should use default prompt and still generate correct output
		assert.Equal(t, 5, len(response.Posts), "Should still generate 5 posts with default prompt")
		assert.Equal(t, 1, len(response.Articles), "Should still generate 1 article with default prompt")

		// Should use hardcoded default prompt as fallback
		llmCall := llmClient.GetLastCall()
		require.NotNil(t, llmCall)
		
		// Should contain default prompt structure
		assert.Contains(t, llmCall.Prompt, idea.Content, "Should still include idea")
	})
}

// TestAsyncJobRegression tests that async job behavior is maintained
func TestAsyncJobRegression(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should maintain async job lifecycle after refactor", func(t *testing.T) {
		// GIVEN draft generation setup
		userID := primitive.NewObjectID().Hex()
		ideaID := primitive.NewObjectID().Hex()

		// WHEN starting async draft generation
		// This will fail until async jobs are properly maintained
		t.Fatal("implement async job lifecycle maintenance - FAILING IN TDD RED PHASE")

		// Should return job ID immediately
		jobResponse, err := startDraftGenerationJob(userID, ideaID)
		require.NoError(t, err)
		
		assert.NotEmpty(t, jobResponse.JobID, "Should return job ID")
		assert.Contains(t, jobResponse.Message, "started", "Should indicate job started")

		// Should be able to check job status
		status, err := checkJobStatus(jobResponse.JobID)
		require.NoError(t, err)
		
		assert.NotEmpty(t, status.JobID, "Status should have job ID")
		assert.NotEmpty(t, status.Status, "Status should have status value")
		assert.Equal(t, jobResponse.JobID, status.JobID, "Job IDs should match")

		// THEN should eventually complete and provide draft IDs
		// Poll for completion (with timeout)
		maxWait := 30 * time.Second
		pollInterval := 1 * time.Second
		start := time.Now()

		for time.Since(start) < maxWait {
			status, err := checkJobStatus(jobResponse.JobID)
			require.NoError(t, err)

			if status.Status == "completed" {
				assert.Len(t, status.DraftIDs, 6, "Should have 6 draft IDs (5 posts + 1 article)")
				return
			}
			
			if status.Status == "failed" {
				assert.Fail(t, "Job should not fail", "Job error: %s", status.Error)
				return
			}

			// Continue polling
			time.Sleep(pollInterval)
		}

		assert.Fail(t, "Job should complete within timeout")
	})
}

// Helper functions for testing
func containsSpanishWords(text string) bool {
	// Simple check for Spanish words
	spanishWords := []string{"el", "la", "de", "que", "y", "en", "un", "es", "se", "no", "te", "lo", "le", "da", "su", "por", "son", "con", "para", "como", "las", "del", "los", "una", "todo", "pero", "más", "ni", "yo", "ya", "me", "si", "bien", "mi", "sí", "tu"}
	
	textLower := strings.ToLower(text)
	for _, word := range spanishWords {
		if strings.Contains(textLower, " "+word+" ") || 
		   strings.HasPrefix(textLower, word+" ") || 
		   strings.HasSuffix(textLower, " "+word) ||
		   textLower == word {
			return true
		}
	}
	return false
}

func isProfessionalTone(text string) bool {
	professionalWords := []string{"profesional", "experiencia", "estrategia", "implementación", "optimización", "análisis", "desarrollo", "tecnología"}
	textLower := strings.ToLower(text)
	for _, word := range professionalWords {
		if strings.Contains(textLower, word) {
			return true
		}
	}
	return false
}

func isCasualTone(text string) bool {
	// Casual tone might be harder to detect, but let's check for certain patterns
	return !strings.Contains(strings.ToLower(text), "profesional") && 
		   strings.Contains(strings.ToLower(text), "vamos") ||
		   strings.Contains(strings.ToLower(text), "hablemos")
}

func isTechnicalTone(text string) bool {
	technicalWords := []string{"algoritmo", "arquitectura", "rendimiento", "optimización", "métrica", "escalabilidad", "implementación"}
	textLower := strings.ToLower(text)
	techCount := 0
	for _, word := range technicalWords {
		if strings.Contains(textLower, word) {
			techCount++
		}
	}
	return techCount >= 2
}

// Setup function that will be implemented
func setupDraftUseCaseWithClients(llmClient interface{}) *usecases.GenerateDraftsUseCase {
	t.Fatal("not implemented - FAILING IN TDD RED PHASE")
	return nil
}

// Test job functions that will be implemented
func startDraftGenerationJob(userID string, ideaID string) (*JobResponse, error) {
	// This will fail until async job implementation is maintained
	t.Fatal("implement async draft generation job - FAILING IN TDD RED PHASE")
	return nil, nil
}

func checkJobStatus(jobID string) (*JobStatusResponse, error) {
	// This will fail until job status checking is maintained
	t.Fatal("implement job status checking - FAILING IN TDD RED PHASE")
	return nil, nil
}

// Response types
type JobResponse struct {
	Message string `json:"message"`
	JobID   string `json:"job_id"`
}

type JobStatusResponse struct {
	JobID      string    `json:"job_id"`
	Status     string    `json:"status"`
	IdeaID     string    `json:"idea_id"`
	DraftIDs   []string  `json:"draft_ids,omitempty"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	StartedAt  time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}
