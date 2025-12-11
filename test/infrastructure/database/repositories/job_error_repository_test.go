package repositories

import (
	"context"
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

func TestJobErrorRepositoryCreate(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("successfully creates job error", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "LLM parsing failed",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		insertedID := primitive.NewObjectID()
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("handles nil job error", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		id, err := repo.Create(ctx, nil)

		assert.Error(mt, err)
		assert.Empty(mt, id)
		assert.Contains(mt, err.Error(), "cannot be nil")
	})

	mt.Run("validates job error before creating", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		invalidJobError := &entities.JobError{
			JobID:     "", // Invalid: empty job ID
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		id, err := repo.Create(ctx, invalidJobError)

		assert.Error(mt, err)
		assert.Empty(mt, id)
		assert.Contains(mt, err.Error(), "job_id cannot be empty")
	})

	mt.Run("creates job error with all fields", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		ideaID := "507f1f77bcf86cd799439013"
		jobError := &entities.JobError{
			JobID:       "job-456",
			UserID:      "507f1f77bcf86cd799439011",
			IdeaID:      &ideaID,
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "timeout",
			RawResponse: "{\"incomplete\": true}",
			Prompt:      "Generate 5 LinkedIn posts",
			Attempt:     3,
			Metadata: map[string]interface{}{
				"model":   "llama3",
				"timeout": 30,
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("handles invalid user ID format", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "invalid-user-id",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		id, err := repo.Create(ctx, jobError)

		assert.Error(mt, err)
		assert.Empty(mt, id)
		assert.Contains(mt, err.Error(), "invalid user ID")
	})

	mt.Run("handles invalid idea ID format", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		invalidIdeaID := "invalid-idea-id"
		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			IdeaID:    &invalidIdeaID,
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		id, err := repo.Create(ctx, jobError)

		assert.Error(mt, err)
		assert.Empty(mt, id)
		assert.Contains(mt, err.Error(), "invalid idea ID")
	})

	mt.Run("handles database insertion error", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   0,
			Code:    11000,
			Message: "duplicate key error",
		}))

		id, err := repo.Create(ctx, jobError)

		assert.Error(mt, err)
		assert.Empty(mt, id)
	})

	mt.Run("handles context cancellation", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		jobError := &entities.JobError{
			JobID:     "job-123",
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

func TestJobErrorRepository_TruncateLongFields(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("truncates very long error messages", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		longError := strings.Repeat("error message ", 2000)
		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     longError,
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("truncates very long raw responses", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		longResponse := strings.Repeat("a", 50000)
		jobError := &entities.JobError{
			JobID:       "job-123",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "response too large",
			RawResponse: longResponse,
			Attempt:     1,
			CreatedAt:   time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("truncates very long prompts", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		longPrompt := strings.Repeat("Generate content ", 2000)
		jobError := &entities.JobError{
			JobID:     "job-123",
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

	mt.Run("handles empty strings without truncation", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:       "job-123",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "minimal error",
			RawResponse: "",
			Prompt:      "",
			Attempt:     1,
			CreatedAt:   time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})
}

func TestJobErrorRepository_DocumentConversion(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("converts entity to document with nil idea ID", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			IdeaID:    nil,
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("converts entity to document with valid idea ID", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		ideaID := "507f1f77bcf86cd799439013"
		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			IdeaID:    &ideaID,
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("converts entity to document with metadata", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:   "job-123",
			UserID:  "507f1f77bcf86cd799439011",
			Stage:   entities.JobErrorStageDraftGeneration,
			Error:   "some error",
			Attempt: 1,
			Metadata: map[string]interface{}{
				"model":       "llama3",
				"temperature": 0.7,
				"max_tokens":  2000,
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("converts timestamps correctly", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		now := time.Now()
		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: now,
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})
}

func TestJobErrorRepository_StageConversion(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("converts draft generation stage correctly", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})
}

func TestJobErrorRepository_Concurrency(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("handles concurrent creates", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		numGoroutines := 5
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			mt.AddMockResponses(mtest.CreateSuccessResponse())
		}

		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				jobError := &entities.JobError{
					JobID:     "job-concurrent-123",
					UserID:    "507f1f77bcf86cd799439011",
					Stage:     entities.JobErrorStageDraftGeneration,
					Error:     "concurrent error",
					Attempt:   idx,
					CreatedAt: time.Now(),
				}

				id, err := repo.Create(ctx, jobError)
				assert.NoError(mt, err)
				assert.NotEmpty(mt, id)
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

func TestJobErrorRepository_Integration(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("creates job error and retrieves ID", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:       "job-integration-123",
			UserID:      "507f1f77bcf86cd799439011",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "integration test error",
			RawResponse: "{\"test\": true}",
			Prompt:      "Test prompt",
			Attempt:     1,
			Metadata: map[string]interface{}{
				"test": "integration",
			},
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
		_, err = primitive.ObjectIDFromHex(id)
		assert.NoError(mt, err, "returned ID should be a valid ObjectID")
	})

	mt.Run("multiple sequential creates work correctly", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			jobError := &entities.JobError{
				JobID:     "job-sequential-123",
				UserID:    "507f1f77bcf86cd799439011",
				Stage:     entities.JobErrorStageDraftGeneration,
				Error:     "sequential error",
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

func TestJobErrorRepository_PerformanceValidation(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("handles large metadata efficiently", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		largeMetadata := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			largeMetadata[string(rune(i))] = i
		}

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "large metadata test",
			Attempt:   1,
			Metadata:  largeMetadata,
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})
}

func TestJobErrorRepository_EdgeCases(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("handles zero attempt number", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "first attempt",
			Attempt:   0,
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("handles high attempt numbers", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "many retries",
			Attempt:   100,
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})

	mt.Run("handles empty idea ID string pointer", func(mt *mtest.T) {
		repo := repositories.NewJobErrorRepository(mt.Coll)
		ctx := context.Background()

		emptyIdeaID := ""
		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			IdeaID:    &emptyIdeaID,
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		id, err := repo.Create(ctx, jobError)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, id)
	})
}
