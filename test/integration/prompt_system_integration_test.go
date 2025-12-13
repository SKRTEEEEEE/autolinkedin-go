package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NOTE: These tests are written according to TDD Red pattern - they will FAIL
// until the complete prompt system integration implementation exists

// TestPromptSystemEndToEnd tests the complete prompt system flow from seed files to processed prompts
func TestPromptSystemEndToEnd(t *testing.T) {
	// This test covers the complete flow:
	// 1. Load prompts from seed files
	// 2. Parse front-matter and content
	// 3. Store in database
	// 4. Process with variable substitution
	// 5. Cache processed prompts
	// 6. Fallback to defaults when needed

	// Setup test environment
	ctx := context.Background()
	tempDir := t.TempDir()
	
	// Create test seed files
	createTestSeedFiles(t, tempDir)
	
	// Create temporary database for testing
	dbName := "test_prompt_system_" + primitive.NewObjectID().Hex()
	testDB := setupTestDB(t, dbName)
	defer testDB.Disconnect(ctx)
	
	// Setup repositories
	promptsRepo := testDB.PromptsRepository()
	
	// Setup services
	logger := &mockLogger{}
	promptLoader := services.NewPromptLoader(logger)
	promptEngine := services.NewPromptEngine(promptsRepo, logger)
	
	t.Run("should load and process complete prompt system", func(t *testing.T) {
		// 1. Load prompts from seed files
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)
		assert.Len(t, promptFiles, 2) // base1.idea.md and pro.draft.md in our test data
		
		// Verify base1 prompt was loaded correctly
		base1Prompt := findPromptFileByName(t, promptFiles, "base1")
		require.NotNil(t, base1Prompt)
		assert.Equal(t, string(entities.PromptTypeIdeas), base1Prompt.Type)
		assert.Contains(t, base1Prompt.PromptTemplate, "{name}")
		assert.Contains(t, base1Prompt.PromptTemplate, "{ideas}")
		assert.Contains(t, base1Prompt.PromptTemplate, "{[related_topics]}")
		
		// 2. Create entities from loaded files
		userID := "test-user-123"
		prompts, err := promptLoader.CreatePromptsFromFile(userID, promptFiles)
		require.NoError(t, err)
		assert.Len(t, prompts, 2)
		
		// 3. Store in database
		for _, prompt := range prompts {
			id, err := promptsRepo.Create(ctx, prompt)
			require.NoError(t, err)
			assert.NotEmpty(t, id)
		}
		
		// Verify prompts were stored
		storedPrompts, err := promptsRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, storedPrompts, 2)
		
		// 4. Setup test data for processing
		now := time.Now()
		testUser := &entities.User{
			ID:         userID,
			Email:      "test@example.com",
			Language:   "es",
			Configuration: map[string]interface{}{
				"name":               "Juan García",
				"expertise":          "Desarrollo Backend",
				"tone_preference":    "Profesional",
				"industry":           "Technology",
				"role":               "Software Engineer",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
		
		testTopic := &entities.Topic{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "Microservicios en Go",
			Ideas:          3,
			RelatedTopics:  []string{"Go", "Arquitectura", "Docker"},
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		
		// 5. Process ideas prompt with variable substitution
		processedPrompt, err := promptEngine.ProcessPrompt(
			ctx,
			userID,
			"base1",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)
		
		// Verify variable substitution worked
		assert.Contains(t, processedPrompt, "Juan García")
		assert.Contains(t, processedPrompt, "Microservicios en Go")
		assert.Contains(t, processedPrompt, "3")
		assert.Contains(t, processedPrompt, "Go, Arquitectura, Docker")
		assert.NotContains(t, processedPrompt, "{name}")
		assert.NotContains(t, processedPrompt, "{ideas}")
		assert.NotContains(t, processedPrompt, "{[related_topics]}")
		assert.NotContains(t, processedPrompt, "{user_context}")
		
		// 6. Test caching by calling ProcessPrompt again
		cachedPrompt, err := promptEngine.ProcessPrompt(
			ctx,
			userID,
			"base1",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)
		assert.Equal(t, processedPrompt, cachedPrompt)
		
		// Verify cache was hit
		assert.True(t, promptEngine.CacheSize() > 0)
		
		// 7. Test draft prompt processing
		testIdea := &entities.Idea{
			ID:          primitive.NewObjectID().Hex(),
			UserID:      userID,
			TopicID:     testTopic.ID,
			Content:     "Cómo implementar circuit breakers en microservicios usando Go",
			Active:      true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		
		draftPrompt, err := promptEngine.ProcessPrompt(
			ctx,
			userID,
			"profesional",
			entities.PromptTypeDrafts,
			nil,
			testIdea,
			testUser,
		)
		require.NoError(t, err)
		
		// Verify variable substitution for draft
		assert.Contains(t, draftPrompt, "Juan García")
		assert.Contains(t, draftPrompt, "Cómo implementar circuit breakers en microservicios usando Go")
		assert.NotContains(t, draftPrompt, "{content}")
		assert.NotContains(t, draftPrompt, "{user_context}")
	})
	
	t.Run("should fallback to default prompts when custom not found", func(t *testing.T) {
		// Try to process a prompt that doesn't exist in database
		// Should fallback to default prompt
		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Test Topic",
			Ideas:         5,
			RelatedTopics: []string{"Topic1", "Topic2"},
			Active:        true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		
		testUser := &entities.User{
			ID:           userID,
			Email:        "test@example.com",
			Configuration: map[string]interface{}{
				"name": "Test User",
			},
		}
		
		processedPrompt, err := promptEngine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		require.NoError(t, err)
		
		// Should contain template variables that were substituted
		assert.Contains(t, processedPrompt, "Test Topic")
		assert.Contains(t, processedPrompt, "5")
		assert.Contains(t, processedPrompt, "Topic1, Topic2")
		assert.Contains(t, processedPrompt, "Test User")
	})
}

// TestPromptSystemSynchronization tests synchronization of seed files with database
func TestPromptSystemSynchronization(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	
	// Initial seed files
	createTestSeedFiles(t, tempDir)
	
	// Setup database
	dbName := "test_sync_" + primitive.NewObjectID().Hex()
	testDB := setupTestDB(t, dbName)
	defer testDB.Disconnect(ctx)
	
	promptsRepo := testDB.PromptsRepository()
	logger := &mockLogger{}
	promptLoader := services.NewPromptLoader(logger)
	
	userID := "user-sync-test"
	
	t.Run("should sync initial seed files to database", func(t *testing.T) {
		// Load from seed
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)
		
		prompts, err := promptLoader.CreatePromptsFromFile(userID, promptFiles)
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
	})
	
	t.Run("should detect changes in seed files", func(t *testing.T) {
		// Create a new seed file
		newSeedFile := filepath.Join(tempDir, "advanced.idea.md")
		newSeedContent := `---
name: advanced
type: ideas
---
Estrategias avanzadas para {name}.
Genera exactamente {ideas} ideas considerando: {[related_topics]}.
Contexto: {user_context}`
		err := os.WriteFile(newSeedFile, []byte(newSeedContent), 0644)
		require.NoError(t, err)
		
		// Reload from seed
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)
		assert.Len(t, promptFiles, 3) // Including new file
		
		// Should detect new file
		newPrompt := findPromptFileByName(t, promptFiles, "advanced")
		require.NotNil(t, newPrompt)
		assert.Equal(t, "advanced", newPrompt.Name)
		assert.Equal(t, string(entities.PromptTypeIdeas), newPrompt.Type)
	})
}

// TestPromptSystemValidation tests validation of prompt syntax and variables
func TestPromptSystemValidation(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	
	// Create invalid seed files for testing
	createInvalidSeedFiles(t, tempDir)
	
	logger := &mockLogger{}
	promptLoader := services.NewPromptLoader(logger)
	
	t.Run("should validate seed file format", func(t *testing.T) {
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)
		
		// Should only load valid files, skip invalid ones
		validFiles := 0
		for _, file := range promptFiles {
			if file.Name != "" {
				validFiles++
			}
		}
		
		// We expect only valid files to be loaded
		assert.Less(t, validFiles, 3)
	})
	
	t.Run("should validate prompt templates", func(t *testing.T) {
		// Create prompts from valid files only
		prompts, err := promptLoader.CreatePromptsFromFile("test-user", promptFiles)
		require.NoError(t, err)
		
		// Verify all created prompts are valid
		for _, prompt := range prompts {
			assert.NotEmpty(t, prompt.ID)
			assert.NotEmpty(t, prompt.UserID)
			assert.NotEmpty(t, prompt.Name)
			assert.NotEmpty(t, prompt.PromptTemplate)
			assert.True(t, prompt.Active)
		}
	})
}

// Helper functions for test setup
func createTestSeedFiles(t *testing.T, dir string) {
	files := map[string]string{
		"base1.idea.md": `---
name: base1
type: ideas
---
Eres un experto en estrategia de contenido para LinkedIn.
Genera {ideas} ideas sobre: {name}
Temas relacionados: {[related_topics]}
Contexto: {user_context}
Devuelve en formato JSON.`,
		"pro.draft.md": `---
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

func createInvalidSeedFiles(t *testing.T, dir string) {
	files := map[string]string{
		"invalid-type.md": `---
name: bad
type: invalid_type
---
This has invalid type.`,
		"missing-frontmatter.md": `This file has no front-matter.`,
		"missing-name.md": `---
type: ideas
---
This is missing the name field.`,
	}
	
	for filename, content := range files {
		path := filepath.Join(dir, filename)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}
}

func findPromptFileByName(t *testing.T, files []*services.PromptFile, name string) *services.PromptFile {
	for _, file := range files {
		if file.Name == name {
			return file
		}
	}
	return nil
}

// Mock implementations
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
func (m *mockLogger) Info(msg string, fields ...interface{})  {}
func (m *mockLogger) Warn(msg string, fields ...interface{})  {}
func (m *mockLogger) Error(msg string, fields ...interface{}) {}
