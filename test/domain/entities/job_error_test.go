package entities_test

import (
	"strings"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/domain/entities"
)

func TestJobError_Creation(t *testing.T) {
	t.Run("valid job error creation", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "failed to parse LLM response",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if jobError.JobID != "job-123" {
			t.Errorf("Expected JobID 'job-123', got '%s'", jobError.JobID)
		}
		if jobError.Stage != entities.JobErrorStageDraftGeneration {
			t.Errorf("Expected Stage 'draft_generation', got '%s'", jobError.Stage)
		}
	})

	t.Run("job error with optional fields", func(t *testing.T) {
		ideaID := "507f1f77bcf86cd799439013"
		jobError := &entities.JobError{
			ID:          "507f1f77bcf86cd799439011",
			JobID:       "job-456",
			UserID:      "507f1f77bcf86cd799439012",
			IdeaID:      &ideaID,
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "LLM timeout",
			RawResponse: "{\"incomplete\": true}",
			Prompt:      "Generate 5 LinkedIn posts about AI",
			Attempt:     2,
			Metadata: map[string]interface{}{
				"model":   "llama3",
				"timeout": 30,
			},
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.NotNil(t, jobError.IdeaID)
		assert.Equal(t, ideaID, *jobError.IdeaID)
		assert.NotEmpty(t, jobError.RawResponse)
		assert.NotEmpty(t, jobError.Prompt)
		assert.Contains(t, jobError.Metadata, "model")
	})

	t.Run("job error with auto-generated timestamp", func(t *testing.T) {
		before := time.Now()
		jobError := &entities.JobError{
			ID:      "507f1f77bcf86cd799439011",
			JobID:   "job-789",
			UserID:  "507f1f77bcf86cd799439012",
			Stage:   entities.JobErrorStageDraftGeneration,
			Error:   "parsing failed",
			Attempt: 1,
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.False(t, jobError.CreatedAt.IsZero())
		assert.True(t, jobError.CreatedAt.After(before) || jobError.CreatedAt.Equal(before))
	})
}

func TestJobError_Validation(t *testing.T) {
	t.Run("nil job error", func(t *testing.T) {
		var jobError *entities.JobError
		err := jobError.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("empty job ID", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job_id cannot be empty")
	})

	t.Run("whitespace-only job ID", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "   ",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job_id cannot be empty")
	})

	t.Run("empty user ID", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id cannot be empty")
	})

	t.Run("whitespace-only user ID", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "   ",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id cannot be empty")
	})

	t.Run("empty stage", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     "",
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stage cannot be empty")
	})

	t.Run("empty error message", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "description cannot be empty")
	})

	t.Run("whitespace-only error message", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "   ",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "description cannot be empty")
	})

	t.Run("negative attempt number", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   -1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "attempt must be >= 0")
	})

	t.Run("zero attempt is valid", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   0,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
	})
}

func TestJobError_Stage(t *testing.T) {
	t.Run("draft generation stage constant", func(t *testing.T) {
		assert.Equal(t, entities.JobErrorStage("draft_generation"), entities.JobErrorStageDraftGeneration)
	})

	t.Run("valid stage assignment", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		assert.Equal(t, entities.JobErrorStageDraftGeneration, jobError.Stage)
		err := jobError.Validate()
		assert.NoError(t, err)
	})
}

func TestJobError_Metadata(t *testing.T) {
	t.Run("metadata can store arbitrary data", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:      "507f1f77bcf86cd799439011",
			JobID:   "job-123",
			UserID:  "507f1f77bcf86cd799439012",
			Stage:   entities.JobErrorStageDraftGeneration,
			Error:   "some error",
			Attempt: 1,
			Metadata: map[string]interface{}{
				"model":        "llama3",
				"temperature":  0.7,
				"max_tokens":   2000,
				"retry_count":  3,
				"error_code":   "TIMEOUT",
				"request_size": 1024,
			},
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.NotNil(t, jobError.Metadata)
		assert.Len(t, jobError.Metadata, 6)
		assert.Equal(t, "llama3", jobError.Metadata["model"])
		assert.Equal(t, 0.7, jobError.Metadata["temperature"])
		assert.Equal(t, 3, jobError.Metadata["retry_count"])
	})

	t.Run("nil metadata is valid", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			Metadata:  nil,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty metadata map is valid", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			Metadata:  map[string]interface{}{},
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.NotNil(t, jobError.Metadata)
		assert.Len(t, jobError.Metadata, 0)
	})
}

func TestJobError_LLMResponseCapture(t *testing.T) {
	t.Run("capture raw LLM response", func(t *testing.T) {
		rawResponse := `{"posts": [{"title": "Post 1"}]}`
		jobError := &entities.JobError{
			ID:          "507f1f77bcf86cd799439011",
			JobID:       "job-123",
			UserID:      "507f1f77bcf86cd799439012",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "incomplete response",
			RawResponse: rawResponse,
			Attempt:     1,
			CreatedAt:   time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.Equal(t, rawResponse, jobError.RawResponse)
	})

	t.Run("capture prompt used for generation", func(t *testing.T) {
		prompt := "Generate 5 LinkedIn posts about Clean Architecture in Go"
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "timeout waiting for response",
			Prompt:    prompt,
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.Equal(t, prompt, jobError.Prompt)
	})

	t.Run("capture both prompt and raw response", func(t *testing.T) {
		prompt := "Generate drafts"
		rawResponse := "{incomplete json"
		jobError := &entities.JobError{
			ID:          "507f1f77bcf86cd799439011",
			JobID:       "job-123",
			UserID:      "507f1f77bcf86cd799439012",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "JSON parsing failed",
			RawResponse: rawResponse,
			Prompt:      prompt,
			Attempt:     1,
			CreatedAt:   time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.Equal(t, prompt, jobError.Prompt)
		assert.Equal(t, rawResponse, jobError.RawResponse)
	})

	t.Run("handle very long raw responses", func(t *testing.T) {
		longResponse := strings.Repeat("a", 50000)
		jobError := &entities.JobError{
			ID:          "507f1f77bcf86cd799439011",
			JobID:       "job-123",
			UserID:      "507f1f77bcf86cd799439012",
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "response too large",
			RawResponse: longResponse,
			Attempt:     1,
			CreatedAt:   time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.Equal(t, 50000, len(jobError.RawResponse))
	})
}

func TestJobError_Timestamps(t *testing.T) {
	t.Run("created_at is set automatically if zero", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Time{},
		}

		before := time.Now()
		err := jobError.Validate()
		after := time.Now()

		assert.NoError(t, err)
		assert.False(t, jobError.CreatedAt.IsZero())
		assert.True(t, jobError.CreatedAt.After(before) || jobError.CreatedAt.Equal(before))
		assert.True(t, jobError.CreatedAt.Before(after) || jobError.CreatedAt.Equal(after))
	})

	t.Run("explicit created_at is preserved", func(t *testing.T) {
		explicitTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: explicitTime,
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.Equal(t, explicitTime, jobError.CreatedAt)
	})
}

func TestJobError_IdeaIDOptional(t *testing.T) {
	t.Run("nil idea ID is valid", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			IdeaID:    nil,
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.Nil(t, jobError.IdeaID)
	})

	t.Run("non-nil idea ID is valid", func(t *testing.T) {
		ideaID := "507f1f77bcf86cd799439013"
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			IdeaID:    &ideaID,
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		require.NotNil(t, jobError.IdeaID)
		assert.Equal(t, ideaID, *jobError.IdeaID)
	})
}

func TestJobError_AttemptTracking(t *testing.T) {
	t.Run("track first attempt", func(t *testing.T) {
		jobError := &entities.JobError{
			ID:        "507f1f77bcf86cd799439011",
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439012",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "first failure",
			Attempt:   0,
			CreatedAt: time.Now(),
		}

		err := jobError.Validate()
		assert.NoError(t, err)
		assert.Equal(t, 0, jobError.Attempt)
	})

	t.Run("track multiple retry attempts", func(t *testing.T) {
		attempts := []int{1, 2, 3, 5, 10}
		for _, attempt := range attempts {
			jobError := &entities.JobError{
				ID:        "507f1f77bcf86cd799439011",
				JobID:     "job-123",
				UserID:    "507f1f77bcf86cd799439012",
				Stage:     entities.JobErrorStageDraftGeneration,
				Error:     "retry failure",
				Attempt:   attempt,
				CreatedAt: time.Now(),
			}

			err := jobError.Validate()
			assert.NoError(t, err, "attempt %d should be valid", attempt)
			assert.Equal(t, attempt, jobError.Attempt)
		}
	})
}
