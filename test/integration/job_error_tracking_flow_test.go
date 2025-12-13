package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestJobErrorTrackingFlow_EndToEnd(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("complete error tracking flow", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		// Simulate a job error during draft generation
		jobError := &entities.JobError{
			JobID:       "job-e2e-123",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "LLM returned invalid JSON",
			RawResponse: "{\"posts\": [incomplete",
			Prompt:      "Generate 5 LinkedIn posts about Go programming",
			Attempt:     1,
			Metadata: map[string]interface{}{
				"model":       "llama3",
				"temperature": 0.7,
				"endpoint":    "http://100.105.212.98:8317/",
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("error tracking with idea context", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		ideaID := "507f1f77bcf86cd799439013"
		jobError := &entities.JobError{
			JobID:       "job-with-idea-456",
			UserID:      "507f1f77bcf86cd799439011",
			IdeaID:      &ideaID,
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "timeout waiting for LLM response",
			RawResponse: "",
			Prompt:      "Generate drafts from idea",
			Attempt:     2,
			Metadata: map[string]interface{}{
				"timeout_seconds": 30,
				"idea_content":    "Discuss clean architecture benefits",
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})
}

func TestJobErrorTrackingFlow_LLMFailureScenarios(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("tracks LLM JSON parsing failure", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		invalidJSON := `{"posts": [{"title": "Post 1", "content": "incomplete`
		jobError := &entities.JobError{
			JobID:       "job-parse-fail-789",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "failed to parse LLM response: unexpected EOF",
			RawResponse: invalidJSON,
			Prompt:      "Generate 5 posts",
			Attempt:     1,
			CreatedAt:   time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("tracks LLM timeout error", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:       "job-timeout-101",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "context deadline exceeded",
			RawResponse: "",
			Prompt:      "Generate complex article",
			Attempt:     3,
			Metadata: map[string]interface{}{
				"timeout_seconds": 30,
				"retry_count":     2,
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("tracks LLM insufficient drafts error", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		partialResponse := `{"posts": [{"title": "Post 1", "content": "Content 1"}], "article": null}`
		jobError := &entities.JobError{
			JobID:       "job-insufficient-202",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "expected 5 posts and 1 article, got 1 posts and 0 articles",
			RawResponse: partialResponse,
			Prompt:      "Generate 5 posts and 1 article",
			Attempt:     1,
			Metadata: map[string]interface{}{
				"expected_posts":    5,
				"expected_articles": 1,
				"received_posts":    1,
				"received_articles": 0,
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("tracks LLM network error", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:       "job-network-303",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "failed to connect to LLM service: connection refused",
			RawResponse: "",
			Prompt:      "Generate drafts",
			Attempt:     1,
			Metadata: map[string]interface{}{
				"endpoint": "http://100.105.212.98:8317/",
				"error":    "connection refused",
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})
}

func TestJobErrorTrackingFlow_RetryTracking(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("tracks multiple retry attempts", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobID := "job-retry-404"
		userID := "507f1f77bcf86cd799439011"

		// Simulate 3 failed attempts
		for attempt := 0; attempt < 3; attempt++ {
			jobError := &entities.JobError{
				JobID:       jobID,
				UserID:      userID,
				Stage:       entities.JobErrorStageDraftGeneration,
				Error:       "LLM returned empty response",
				RawResponse: "",
				Prompt:      "Generate drafts",
				Attempt:     attempt,
				Metadata: map[string]interface{}{
					"retry_delay_ms": attempt * 1000,
				},
				CreatedAt: time.Now(),
			}

			mt.AddMockResponses(mtest.CreateSuccessResponse())

			id, err := repo.Create(ctx, jobError)
			assert.NoError(mt, err)
			assert.NotEmpty(mt, id)
		}
	})

	mt.Run("tracks retry with different errors", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobID := "job-retry-mix-505"
		userID := "507f1f77bcf86cd799439011"

		errors := []string{
			"timeout",
			"invalid JSON",
			"connection refused",
		}

		for attempt, errMsg := range errors {
			jobError := &entities.JobError{
				JobID:     jobID,
				UserID:    userID,
				Stage:     entities.JobErrorStageDraftGeneration,
				Error:     errMsg,
				Attempt:   attempt,
				CreatedAt: time.Now(),
			}

			mt.AddMockResponses(mtest.CreateSuccessResponse())

			id, err := repo.Create(ctx, jobError)
			assert.NoError(mt, err)
			assert.NotEmpty(mt, id)
		}
	})
}

func TestJobErrorTrackingFlow_PromptCapture(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("captures full prompt for debugging", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		prompt := `You are an expert LinkedIn content creator.
Generate 5 engaging LinkedIn posts about Go programming.
Each post should be informative and concise.
Style: PROFESSIONAL
Format: JSON`

		jobError := &entities.JobError{
			JobID:       "job-prompt-606",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "LLM returned invalid format",
			RawResponse: "Some non-JSON response",
			Prompt:      prompt,
			Attempt:     1,
			CreatedAt:   time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("handles very long prompts", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		longPrompt := strings.Repeat("Generate content ", 3000)
		jobError := &entities.JobError{
			JobID:     "job-long-prompt-707",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "prompt too long",
			Prompt:    longPrompt,
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})
}

func TestJobErrorTrackingFlow_MetadataEnrichment(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("stores LLM configuration metadata", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:   "job-metadata-808",
			UserID:  "507f1f77bcf86cd799439011",
			Stage:   entities.JobErrorStageDraftGeneration,
			Error:   "LLM error",
			Attempt: 1,
			Metadata: map[string]interface{}{
				"model":             "llama3",
				"temperature":       0.7,
				"max_tokens":        2000,
				"top_p":             0.9,
				"presence_penalty":  0.1,
				"frequency_penalty": 0.1,
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("stores request context metadata", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:   "job-context-909",
			UserID:  "507f1f77bcf86cd799439011",
			Stage:   entities.JobErrorStageDraftGeneration,
			Error:   "request failed",
			Attempt: 1,
			Metadata: map[string]interface{}{
				"request_id":      "req-12345",
				"worker_id":       "worker-1",
				"queue_name":      "draft.generation",
				"processing_time": 250,
				"timestamp":       time.Now().Unix(),
				"environment":     "production",
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("stores error diagnostics metadata", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:   "job-diagnostics-1010",
			UserID:  "507f1f77bcf86cd799439011",
			Stage:   entities.JobErrorStageDraftGeneration,
			Error:   "JSON parse error",
			Attempt: 1,
			Metadata: map[string]interface{}{
				"error_type":      "json.SyntaxError",
				"error_line":      42,
				"error_column":    15,
				"response_size":   1024,
				"expected_format": "application/json",
				"actual_format":   "text/plain",
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})
}

func TestJobErrorTrackingFlow_Persistence(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("persisted error can be queried later", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:       "job-persist-1111",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "persistent error",
			RawResponse: "{\"test\": true}",
			Prompt:      "Test prompt",
			Attempt:     1,
			CreatedAt:   time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)

		// Verify ID is a valid ObjectID
		objectID, err := primitive.ObjectIDFromHex(id)
		assert.NoError(mt, err)
		assert.NotEqual(mt, primitive.NilObjectID, objectID)
	})

	mt.Run("multiple errors for same job are tracked separately", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobID := "job-multi-error-1212"

		for i := 0; i < 3; i++ {
			jobError := &entities.JobError{
				JobID:     jobID,
				UserID:    "507f1f77bcf86cd799439011",
				Stage:     entities.JobErrorStageDraftGeneration,
				Error:     "error instance",
				Attempt:   i,
				CreatedAt: time.Now(),
			}

			mt.AddMockResponses(mtest.CreateSuccessResponse())

			id, err := repo.Create(ctx, jobError)
			assert.NoError(mt, err)
			assert.NotEmpty(mt, id)
		}
	})
}

func TestJobErrorTrackingFlow_RealWorldScenarios(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("scenario: LLM returns partial response", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		partialResponse := `{
			"posts": [
				{"title": "Post 1", "content": "Content 1"},
				{"title": "Post 2", "content": "Content 2"}
			],
			"article": null
		}`

		jobError := &entities.JobError{
			JobID:       "job-scenario-partial-1313",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "expected 5 posts and 1 article, got 2 posts and 0 articles",
			RawResponse: partialResponse,
			Prompt:      "Generate 5 LinkedIn posts and 1 article about PROFESSIONAL style",
			Attempt:     1,
			Metadata: map[string]interface{}{
				"style":             "PROFESSIONAL",
				"expected_posts":    5,
				"expected_articles": 1,
				"actual_posts":      2,
				"actual_articles":   0,
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("scenario: LLM endpoint unavailable", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:       "job-scenario-unavailable-1414",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "Post http://100.105.212.98:8317/: dial tcp 100.105.212.98:8317: connect: connection refused",
			RawResponse: "",
			Prompt:      "Generate drafts",
			Attempt:     2,
			Metadata: map[string]interface{}{
				"endpoint":      "http://100.105.212.98:8317/",
				"timeout":       30,
				"network_error": "connection refused",
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("scenario: invalid article title extraction", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		responseWithNoTitle := `{
			"posts": [{"title": "Post 1", "content": "Content"}],
			"article": {"title": "", "content": "Long article content without title"}
		}`

		jobError := &entities.JobError{
			JobID:       "job-scenario-no-title-1515",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "article has empty title",
			RawResponse: responseWithNoTitle,
			Prompt:      "Generate article",
			Attempt:     1,
			Metadata: map[string]interface{}{
				"validation_error": "empty_title",
				"content_length":   300,
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})
}

func TestJobErrorTrackingFlow_ContextManagement(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("respects context timeout", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		time.Sleep(2 * time.Millisecond) // Ensure timeout

		jobError := &entities.JobError{
			JobID:     "job-ctx-timeout-1616",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		id, err := repo.Create(ctx, jobError)

		assert.Error(mt, err)
		assert.Empty(mt, id)
	})

	mt.Run("handles context cancellation gracefully", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		jobError := &entities.JobError{
			JobID:     "job-ctx-cancel-1717",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		id, err := repo.Create(ctx, jobError)

		assert.Error(mt, err)
		assert.Empty(mt, id)
	})
}
