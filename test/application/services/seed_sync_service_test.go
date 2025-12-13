package services

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories"
)

// TestSeedSyncService tests the seed synchronization service
// Ensures seed files and database remain in sync
func TestSeedSyncService(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should synchronize all seed prompts from files to database", func(t *testing.T) {
		// GIVEN a clean database and seed prompt directory
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		promptRepo := repositories.NewPromptRepository(db)

		// Check seed prompt files exist
		seedPromptDir := filepath.Join("..", "..", "..", "seed", "prompt")
		files, err := os.ReadDir(seedPromptDir)
		require.NoError(t, err)
		require.Greater(t, len(files), 0, "Seed prompt directory should contain files")

		// Parse all seed prompt files
		var seedPrompts []entities.Prompt
		for _, file := range files {
			if file.IsDir() || !isPromptFile(file.Name()) {
				continue
			}

			filePath := filepath.Join(seedPromptDir, file.Name())
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)

			var prompt entities.Prompt
			err = json.Unmarshal(content, &prompt)
			require.NoError(t, err)

			// Set required fields for seeding
			prompt.UserID = userID
			prompt.Active = true
			if prompt.CreatedAt.IsZero() {
				prompt.CreatedAt = time.Now()
			}
			if prompt.UpdatedAt.IsZero() {
				prompt.UpdatedAt = time.Now()
			}

			seedPrompts = append(seedPrompts, prompt)
		}

		// WHEN synchronizing prompts
		syncService := NewSeedSyncService(promptRepo)
		err = syncService.SyncPrompts(ctx, userID, seedPrompts)

		// This will fail until the sync service is implemented
		t.Fatal("implement seed prompt synchronization service - FAILING IN TDD RED PHASE")

		// THEN should:
		// 1. Create prompts that don't exist
		// 2. Update prompts that have changed
		// 3. Archive/deactivate prompts that no longer exist in seed

		// Verify all prompts are in database
		dbPrompts, err := promptRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, len(seedPrompts), len(dbPrompts))
	})

	tt.Run("should synchronize all seed topics from topic.json to database", func(t *testing.T) {
		// GIVEN a clean database and seed topic file
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		topicRepo := repositories.NewTopicRepository(db)

		// Load seed topics configuration
		seedTopicFile := filepath.Join("..", "..", "..", "seed", "topic.json")
		content, err := os.ReadFile(seedTopicFile)
		require.NoError(t, err)

		var seedTopics []entities.Topic
		err = json.Unmarshal(content, &seedTopics)
		require.NoError(t, err)

		// Prepare topics with required fields
		for i := range seedTopics {
			seedTopics[i].UserID = userID
			if seedTopics[i].CreatedAt.IsZero() {
				seedTopics[i].CreatedAt = time.Now()
			}
			if seedTopics[i].UpdatedAt.IsZero() {
				seedTopics[i].UpdatedAt = time.Now()
			}
		}

		// WHEN synchronizing topics
		syncService := NewSeedSyncService(topicRepo)
		err = syncService.SyncTopics(ctx, userID, seedTopics)

		// This will fail until topic sync is implemented
		t.Fatal("implement seed topic synchronization service - FAILING IN TDD RED PHASE")

		// THEN should:
		// 1. Create topics that don't exist
		// 2. Update topics that have changed
		// 3. Archive topics that no longer exist
		// 4. Validate that all prompt references exist

		// Verify all topics are in database
		dbTopics, err := topicRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, len(seedTopics), len(dbTopics))
	})

	tt.Run("should validate prompt references before syncing topics", func(t *testing.T) {
		// GIVEN topics with potentially invalid prompt references
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		topicRepo := repositories.NewTopicRepository(db)
		promptRepo := repositories.NewPromptRepository(db)

		// Create a seed topic with non-existent prompt reference
		topicsWithInvalidRefs := []entities.Topic{
			{
				ID:          primitive.NewObjectID().Hex(),
				UserID:      userID,
				Name:        "Invalid Topic",
				Description: "Topic with invalid prompt reference",
				PromptName:  "non-existent-prompt",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		// WHEN attempting to sync with invalid references
		syncService := NewSeedSyncService(topicRepo, promptRepo)
		err := syncService.SyncTopics(ctx, userID, topicsWithInvalidRefs)

		// This will fail until prompt reference validation is implemented
		t.Fatal("implement prompt reference validation - FAILING IN TDD RED PHASE")

		// THEN should return validation error for missing prompt reference
		_ = err // Will be properly asserted after implementation
	})

	tt.Run("should handle sync failure scenarios gracefully", func(t *testing.T) {
		testCases := []struct {
			name           string
			setupScenario  func() error
			expectedError  string
			expectedAction string
		}{
			{
				name: "missing seed directory",
				setupScenario: func() error {
					// Simulate missing seed directory
					// This will fail until directory existence check is implemented
					return nil
				},
				expectedError:  "seed directory not found",
				expectedAction: "stop sync without database changes",
			},
			{
				name: "malformed seed file",
				setupScenario: func() error {
					// Simulate malformed JSON in seed file
					// This will fail until JSON validation is implemented
					return nil
				},
				expectedError:  "invalid seed file format",
				expectedAction: "skip malformed file, continue with others",
			},
			{
				name: "database connection lost during sync",
				setupScenario: func() error {
					// Simulate database connection issue
					// This will fail until connection error handling is implemented
					return nil
				},
				expectedError:  "database connection error",
				expectedAction: "rollback partial changes",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// This will fail until error scenarios are implemented
				t.Fatal("implement sync error handling for: " + tc.name + " - FAILING IN TDD RED PHASE")
			})
		}
	})
}

// TestSeedSyncDifferentialSync tests that sync only updates what has changed
func TestSeedSyncDifferentialSync(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should only update prompts that have changed", func(t *testing.T) {
		// GIVEN existing prompts in database
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		promptRepo := repositories.NewPromptRepository(db)

		// Create initial prompt
		existingPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "test-prompt",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Old template",
			Active:         true,
			CreatedAt:      time.Now().Add(-1 * time.Hour), // Older
			UpdatedAt:      time.Now().Add(-1 * time.Hour),
		}

		err := promptRepo.Create(ctx, existingPrompt)
		require.NoError(t, err)

		// WHEN syncing with updated prompt
		updatedPrompt := *existingPrompt
		updatedPrompt.PromptTemplate = "New updated template"
		updatedPrompt.UpdatedAt = time.Now()

		syncService := NewSeedSyncService(promptRepo)
		err = syncService.SyncPrompts(ctx, userID, []entities.Prompt{updatedPrompt})

		// This will fail until differential sync is implemented
		t.Fatal("implement differential sync logic - FAILING IN TDD RED PHASE")

		// THEN should update only the changed prompt
		// Verify prompt was updated
		updated, err := promptRepo.GetByID(ctx, existingPrompt.ID)
		require.NoError(t, err)
		assert.Equal(t, "New updated template", updated.PromptTemplate)
	})

	tt.Run("should preserve user modifications to active status", func(t *testing.T) {
		// GIVEN prompts with user modifications
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		promptRepo := repositories.NewPromptRepository(db)

		// Create prompt that user has deactivated
		userModifiedPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "user-deactivated",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Template content",
			Active:         false, // User deactivated
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := promptRepo.Create(ctx, userModifiedPrompt)
		require.NoError(t, err)

		// WHEN syncing with active prompt in seed
		seedPrompt := *userModifiedPrompt
		seedPrompt.Active = true // Active in seed

		syncService := NewSeedSyncService(promptRepo)
		err = syncService.SyncPrompts(ctx, userID, []entities.Prompt{seedPrompt})

		// This will fail until user preference preservation is implemented
		t.Fatal("implement user modification preservation during sync - FAILING IN TDD RED PHASE")

		// THEN should preserve user's deactivated status
		// Verify prompt is still deactivated
		unmodified, err := promptRepo.GetByID(ctx, userModifiedPrompt.ID)
		require.NoError(t, err)
		assert.False(t, unmodified.Active, "Should preserve user's deactivated preference")
	})
}

// isPromptFile checks if a filename is a valid prompt file
func isPromptFile(filename string) bool {
	return filename != ".gitkeep" &&
		filename != "README.md"

	// Will be extended to check for .idea.md, .draft.md patterns when implemented
}

// setupTestDB creates a test database connection
// This will fail until the test database setup is implemented
func setupTestDB(t *testing.T) *mongo.Database {
	t.Fatal("test database setup not implemented yet - FAILING IN TDD RED PHASE")
	return nil
}

// cleanupTestDB cleans up the test database
func cleanupTestDB(t *testing.T, db *mongo.Database) {
	if db == nil {
		return
	}
	// Implementation will go here
}

// NewSeedSyncService creates a new seed synchronization service
func NewSeedSyncService(repos ...interface{}) interface{} {
	// This will fail until the service is implemented
	return struct{}{}
}
