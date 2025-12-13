package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NOTE: These tests are written according to TDD Red pattern - they will FAIL
// until the prompt system synchronization implementation exists

func TestPromptSystemSynchronization(t *testing.T) {
	// This test covers synchronization between seed files and database:
	// 1. Detect changes in seed files
	// 2. Update database with new/modified prompts
	// 3. Skip .old.md files
	// 4. Handle deletions when seed files are removed

	// Setup test environment
	ctx := context.Background()
	tempDir := t.TempDir()
	testDB := setupTestDB(t, "test_sync")
	defer testDB.Disconnect(ctx)

	promptsRepo := testDB.PromptsRepository()
	logger := &mockLogger{}

	userID := "sync-test-user"

	t.Run("should detect and sync new seed files", func(t *testing.T) {
		// Create initial seed files
		createInitialSeedFiles(t, tempDir)

		// Load and sync
		loader := NewPromptLoader(logger)
		promptFiles, err := loader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)
		assert.Len(t, promptFiles, 2) // Should skip .old.md files

		prompts, err := loader.CreatePromptsFromFile(userID, promptFiles)
		require.NoError(t, err)

		// Store in database
		for _, prompt := range prompts {
			_, err := promptsRepo.Create(ctx, prompt)
			require.NoError(t, err)
		}

		// Verify stored
		stored, err := promptsRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, stored, 2)

		// Now add a new seed file
		newFile := filepath.Join(tempDir, "advanced.draft.md")
		newContent := `---
name: advanced
type: drafts
---
Advanced template for {content} with context: {user_context}
`

		err = os.WriteFile(newFile, []byte(newContent), 0644)
		require.NoError(t, err)

		// Reload and detect changes
		newPromptFiles, err := loader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)
		assert.Len(t, newPromptFiles, 3)

		// Find the new prompt
		var newPrompt *PromptFile
		for _, pf := range newPromptFiles {
			if pf.Name == "advanced" {
				newPrompt = pf
				break
			}
		}
		require.NotNil(t, newPrompt)
		assert.Equal(t, "advanced", newPrompt.Name)
		assert.Equal(t, string(entities.PromptTypeDrafts), newPrompt.Type)

		// Store the new prompt
		newPrompts, err := loader.CreatePromptsFromFile(userID, []*PromptFile{newPrompt})
		require.NoError(t, err)
		require.Len(t, newPrompts, 1)

		_, err = promptsRepo.Create(ctx, newPrompts[0])
		require.NoError(t, err)

		// Verify database now has 3 prompts
		allPrompts, err := promptsRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, allPrompts, 3)
	})

	t.Run("should detect modified content in seed files", func(t *testing.T) {
		// Get initial prompt count
		initialCount, err := promptsRepo.CountByUserID(ctx, userID)
		require.NoError(t, err)

		// Modify an existing seed file
		existingFile := filepath.Join(tempDir, "base1.idea.md")
		modifiedContent := `---
name: base1
type: ideas
---
MODIFIED TEMPLATE for {name} generating {ideas} ideas.
Topics: {[related_topics]}.
Context: {user_context}.
`

		err = os.WriteFile(existingFile, []byte(modifiedContent), 0644)
		require.NoError(t, err)

		// Reload and verify content changed
		promptFiles, err := loader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)

		base1Prompt := findPromptFileByName(t, promptFiles, "base1")
		require.NotNil(t, base1Prompt)
		assert.Contains(t, base1Prompt.PromptTemplate, "MODIFIED TEMPLATE")

		// Update the prompt in database
		prompts, err := loader.CreatePromptsFromFile(userID, []*PromptFile{base1Prompt})
		require.NoError(t, err)
		require.Len(t, prompts, 1)

		// Find existing prompt by name
		existingPrompt, err := promptsRepo.FindByName(ctx, userID, "base1")
		require.NoError(t, err)
		require.NotNil(t, existingPrompt)

		// Update with new content
		newPrompt := prompts[0]
		newPrompt.ID = existingPrompt.ID // Keep the same ID
		newPrompt.UserID = existingPrompt.UserID
		newPrompt.CreatedAt = existingPrompt.CreatedAt
		newPrompt.UpdatedAt = time.Now()

		err = promptsRepo.Update(ctx, newPrompt)
		require.NoError(t, err)

		// Verify updated content
		updatedPrompt, err := promptsRepo.FindByName(ctx, userID, "base1")
		require.NoError(t, err)
		assert.Contains(t, updatedPrompt.PromptTemplate, "MODIFIED TEMPLATE")

		// Verify count unchanged (just updated)
		finalCount, err := promptsRepo.CountByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, initialCount, finalCount)
	})

	t.Run("should skip legacy .old.md files", func(t *testing.T) {
		// Create a legacy file
		legacyFile := filepath.Join(tempDir, "legacy.idea.old.md")
		legacyContent := `---
name: legacy
type: ideas
---
This should be ignored.
`

		err := os.WriteFile(legacyFile, []byte(legacyContent), 0644)
		require.NoError(t, err)

		// Reload and verify legacy file is skipped
		promptFiles, err := loader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)

		// Should not load the legacy file
		legacyPrompt := findPromptFileByName(t, promptFiles, "legacy")
		assert.Nil(t, legacyPrompt)

		// Count should remain unchanged
		count, err := promptsRepo.CountByUserID(ctx, userID)
		require.NoError(t, err)
		expectedCount := int64(3) // base1, profesional, advanced
		assert.Equal(t, expectedCount, count)
	})
}

func TestPromptSystemSynchronizationWithUsers(t *testing.T) {
	// Test synchronization behavior with multiple users
	ctx := context.Background()
	tempDir := t.TempDir()
	testDB := setupTestDB(t, "test_sync_users")
	defer testDB.Disconnect(ctx)

	promptsRepo := testDB.PromptsRepository()
	logger := &mockLogger{}

	user1ID := "user-1"
	user2ID := "user-2"

	t.Run("should sync seed prompts for different users", func(t *testing.T) {
		// Create seed files
		createInitialSeedFiles(t, tempDir)

		loader := NewPromptLoader(logger)
		promptFiles, err := loader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)

		// Create prompts for user 1
		user1Prompts, err := loader.CreatePromptsFromFile(user1ID, promptFiles)
		require.NoError(t, err)
		require.Len(t, user1Prompts, 2)

		// Store for user 1
		for _, prompt := range user1Prompts {
			_, err := promptsRepo.Create(ctx, prompt)
			require.NoError(t, err)
		}

		// Create prompts for user 2
		user2Prompts, err := loader.CreatePromptsFromFile(user2ID, promptFiles)
		require.NoError(t, err)
		require.Len(t, user2Prompts, 2)

		// Store for user 2
		for _, prompt := range user2Prompts {
			_, err := promptsRepo.Create(ctx, prompt)
			require.NoError(t, err)
		}

		// Verify each user has their own prompts
		user1Stored, err := promptsRepo.ListByUserID(ctx, user1ID)
		require.NoError(t, err)
		assert.Len(t, user1Stored, 2)

		user2Stored, err := promptsRepo.ListByUserID(ctx, user2ID)
		require.NoError(t, err)
		assert.Len(t, user2Stored, 2)

		// Verify prompts are separate (different IDs)
		user1Prompt, err := promptsRepo.FindByName(ctx, user1ID, "base1")
		require.NoError(t, err)

		user2Prompt, err := promptsRepo.FindByName(ctx, user2ID, "base1")
		require.NoError(t, err)

		assert.NotEqual(t, user1Prompt.ID, user2Prompt.ID)
		assert.Equal(t, user1ID, user1Prompt.UserID)
		assert.Equal(t, user2ID, user2Prompt.UserID)
	})

	t.Run("should allow users to modify their prompts independently", func(t *testing.T) {
		// Get user 1's prompt
		user1Prompt, err := promptsRepo.FindByName(ctx, user1ID, "base1")
		require.NoError(t, err)

		// Modify user 1's prompt
		user1Prompt.PromptTemplate = "USER 1 CUSTOM TEMPLATE for {name}"
		err = promptsRepo.Update(ctx, user1Prompt)
		require.NoError(t, err)

		// Verify user 1 changed but user 2 unchanged
		updatedUser1Prompt, err := promptsRepo.FindByName(ctx, user1ID, "base1")
		require.NoError(t, err)
		assert.Contains(t, updatedUser1Prompt.PromptTemplate, "USER 1 CUSTOM")

		user2Prompt, err := promptsRepo.FindByName(ctx, user2ID, "base1")
		require.NoError(t, err)
		assert.NotContains(t, user2Prompt.PromptTemplate, "USER 1 CUSTOM")
	})
}

// Helper functions
func createInitialSeedFiles(t *testing.T, dir string) {
	files := map[string]string{
		"base1.idea.md": `---
name: base1
type: ideas
---
Eres un experto en estrategia de contenido para LinkedIn.
Genera {ideas} ideas sobre: {name}
Temas relacionados: {[related_topics]}
Contexto: {user_context}`,
		"profesional.draft.md": `---
name: profesional
type: drafts
---
Basado en: {content}
Contexto: {user_context}
Genera contenido profesional para LinkedIn.`,
		"base1.old.md": `---
name: base1
type: ideas
---
Este es un archivo antiguo que debe ser ignorado.`,
	}

	for filename, content := range files {
		path := filepath.Join(dir, filename)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}
}

func findPromptFileByName(t *testing.T, files []*PromptFile, name string) *PromptFile {
	for _, file := range files {
		if file.Name == name {
			return file
		}
	}
	return nil
}

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
func (m *mockLogger) Info(msg string, fields ...interface{})  {}
func (m *mockLogger) Warn(msg string, fields ...interface{})  {}
func (m *mockLogger) Error(msg string, fields ...interface{}) {}
