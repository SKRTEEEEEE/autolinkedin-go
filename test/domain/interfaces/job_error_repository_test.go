package interfaces

import (
	"context"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockJobErrorRepository is a mock implementation of JobErrorRepository
type MockJobErrorRepository struct {
	mock.Mock
}

func (m *MockJobErrorRepository) Create(ctx context.Context, jobError *entities.JobError) (string, error) {
	args := m.Called(ctx, jobError)
	return args.String(0), args.Error(1)
}

func TestJobErrorRepositoryInterface(t *testing.T) {
	t.Run("interface has Create method", func(t *testing.T) {
		var repo interfaces.JobErrorRepository
		mockRepo := new(MockJobErrorRepository)
		repo = mockRepo

		assert.NotNil(t, repo)
		assert.Implements(t, (*interfaces.JobErrorRepository)(nil), mockRepo)
	})
}

func TestJobErrorRepository_Create(t *testing.T) {
	t.Run("successfully creates job error", func(t *testing.T) {
		mockRepo := new(MockJobErrorRepository)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "LLM parsing failed",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		expectedID := "507f1f77bcf86cd799439012"
		mockRepo.On("Create", ctx, jobError).Return(expectedID, nil)

		id, err := mockRepo.Create(ctx, jobError)

		assert.NoError(t, err)
		assert.Equal(t, expectedID, id)
		mockRepo.AssertExpectations(t)
	})

	t.Run("handles nil job error", func(t *testing.T) {
		mockRepo := new(MockJobErrorRepository)
		ctx := context.Background()

		mockRepo.On("Create", ctx, (*entities.JobError)(nil)).Return("", assert.AnError)

		id, err := mockRepo.Create(ctx, nil)

		assert.Error(t, err)
		assert.Empty(t, id)
		mockRepo.AssertExpectations(t)
	})

	t.Run("handles invalid job error", func(t *testing.T) {
		mockRepo := new(MockJobErrorRepository)
		ctx := context.Background()

		invalidJobError := &entities.JobError{
			JobID: "", // Invalid: empty job ID
		}

		mockRepo.On("Create", ctx, invalidJobError).Return("", assert.AnError)

		id, err := mockRepo.Create(ctx, invalidJobError)

		assert.Error(t, err)
		assert.Empty(t, id)
		mockRepo.AssertExpectations(t)
	})

	t.Run("propagates context", func(t *testing.T) {
		mockRepo := new(MockJobErrorRepository)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		mockRepo.On("Create", ctx, jobError).Return("507f1f77bcf86cd799439012", nil)

		id, err := mockRepo.Create(ctx, jobError)

		assert.NoError(t, err)
		assert.NotEmpty(t, id)
		mockRepo.AssertExpectations(t)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		mockRepo := new(MockJobErrorRepository)
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

		mockRepo.On("Create", ctx, jobError).Return("", context.Canceled)

		id, err := mockRepo.Create(ctx, jobError)

		assert.Error(t, err)
		assert.Empty(t, id)
		assert.ErrorIs(t, err, context.Canceled)
		mockRepo.AssertExpectations(t)
	})

	t.Run("handles database connection errors", func(t *testing.T) {
		mockRepo := new(MockJobErrorRepository)
		ctx := context.Background()

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		mockRepo.On("Create", ctx, jobError).Return("", assert.AnError)

		id, err := mockRepo.Create(ctx, jobError)

		assert.Error(t, err)
		assert.Empty(t, id)
		mockRepo.AssertExpectations(t)
	})

	t.Run("creates job error with all optional fields", func(t *testing.T) {
		mockRepo := new(MockJobErrorRepository)
		ctx := context.Background()

		ideaID := "507f1f77bcf86cd799439013"
		jobError := &entities.JobError{
			JobID:       "job-456",
			UserID:      "507f1f77bcf86cd799439011",
			IdeaID:      &ideaID,
			Stage:       entities.JobErrorStageDraftGeneration,
			Error:       "timeout",
			RawResponse: "{\"incomplete\": true}",
			Prompt:      "Generate drafts",
			Attempt:     3,
			Metadata: map[string]interface{}{
				"model":   "llama3",
				"timeout": 30,
			},
			CreatedAt: time.Now(),
		}

		expectedID := "507f1f77bcf86cd799439014"
		mockRepo.On("Create", ctx, jobError).Return(expectedID, nil)

		id, err := mockRepo.Create(ctx, jobError)

		assert.NoError(t, err)
		assert.Equal(t, expectedID, id)
		require.NotNil(t, jobError.IdeaID)
		assert.Equal(t, ideaID, *jobError.IdeaID)
		assert.NotEmpty(t, jobError.RawResponse)
		assert.NotEmpty(t, jobError.Prompt)
		assert.NotNil(t, jobError.Metadata)
		mockRepo.AssertExpectations(t)
	})
}

func TestJobErrorRepository_Concurrency(t *testing.T) {
	t.Run("handles concurrent creates", func(t *testing.T) {
		mockRepo := new(MockJobErrorRepository)
		ctx := context.Background()

		numGoroutines := 10
		jobErrors := make([]*entities.JobError, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			jobErrors[i] = &entities.JobError{
				JobID:     "job-concurrent-123",
				UserID:    "507f1f77bcf86cd799439011",
				Stage:     entities.JobErrorStageDraftGeneration,
				Error:     "concurrent error",
				Attempt:   i,
				CreatedAt: time.Now(),
			}

			mockRepo.On("Create", ctx, jobErrors[i]).Return("507f1f77bcf86cd799439012", nil)
		}

		done := make(chan bool, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				id, err := mockRepo.Create(ctx, jobErrors[idx])
				assert.NoError(t, err)
				assert.NotEmpty(t, id)
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		mockRepo.AssertExpectations(t)
	})
}

func TestJobErrorRepository_ContextTimeout(t *testing.T) {
	t.Run("handles context timeout", func(t *testing.T) {
		mockRepo := new(MockJobErrorRepository)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		time.Sleep(2 * time.Millisecond) // Ensure timeout

		jobError := &entities.JobError{
			JobID:     "job-123",
			UserID:    "507f1f77bcf86cd799439011",
			Stage:     entities.JobErrorStageDraftGeneration,
			Error:     "some error",
			Attempt:   1,
			CreatedAt: time.Now(),
		}

		mockRepo.On("Create", ctx, jobError).Return("", context.DeadlineExceeded)

		id, err := mockRepo.Create(ctx, jobError)

		assert.Error(t, err)
		assert.Empty(t, id)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		mockRepo.AssertExpectations(t)
	})
}
