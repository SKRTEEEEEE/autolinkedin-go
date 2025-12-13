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
// until the prompt validation implementation exists

func TestPromptValidation(t *testing.T) {
	// This test covers validation of prompt templates:
	// 1. Validate template syntax
	// 2. Check for required variables based on prompt type
	// 3. Validate front-matter in seed files
	// 4. Skip invalid files gracefully
	// 5. Report validation errors appropriately

	ctx := context.Background()
	tempDir := t.TempDir()

	// Setup test environment
	testDB := setupTestDB(t, "test_validation")
	defer testDB.Disconnect(ctx)

	promptsRepo := testDB.PromptsRepository()
	logger := &mockLogger{}
	promptLoader := NewPromptLoader(logger)

	t.Run("should validate prompt template with required variables", func(t *testing.T) {
		userID := "validation-test-user"

		// Create valid prompt template for ideas
		ideasFile := filepath.Join(tempDir, "valid-ideas.md")
		ideasContent := `---
name: valid-ideas
type: ideas
---
Eres un experto en crear contenido para LinkedIn.
Genera {ideas} ideas sobre: {name}
Temas relacionados: {[related_topics]}
Contexto: {user_context}`

		err := os.WriteFile(ideasFile, []byte(ideasContent), 0644)
		require.NoError(t, err)

		// Load and validate
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)
		require.Len(t, promptFiles, 1)

		// Should have extracted name and type correctly
		ideasPrompt := promptFiles[0]
		assert.Equal(t, "valid-ideas", ideasPrompt.Name)
		assert.Equal(t, string(entities.PromptTypeIdeas), ideasPrompt.Type)
		assert.NotEmpty(t, ideasPrompt.PromptTemplate)

		// Create entities and validate
		prompts, err := promptLoader.CreatePromptsFromFile(userID, promptFiles)
		require.NoError(t, err)
		require.Len(t, prompts, 1)

		// Validate that required variables exist
		ideasEntity := prompts[0]
		assert.Contains(t, ideasEntity.PromptTemplate, "{name}")
		assert.Contains(t, ideasEntity.PromptTemplate, "{ideas}")
		assert.Contains(t, ideasEntity.PromptTemplate, "{[related_topics]}")
		assert.Contains(t, ideasEntity.PromptTemplate, "{user_context}")

		// Entity should be valid
		assert.NoError(t, ideasEntity.Validate())

		// Should be able to store in database
		_, err = promptsRepo.Create(ctx, ideasEntity)
		require.NoError(t, err)
	})

	t.Run("should validate draft prompt template with required variables", func(t *testing.T) {
		// Create valid prompt template for drafts
		draftFile := filepath.Join(tempDir, "valid-draft.md")
		draftContent := `---
name: valid-draft
type: drafts
---
Basado en la idea: {content}
Contexto del usuario: {user_context}
Instrucciones para generar contenido profesional.`

		err := os.WriteFile(draftFile, []byte(draftContent), 0644)
		require.NoError(t, err)

		// Load and validate
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)
		require.Len(t, promptFiles, 1)

		// Should have extracted name and type correctly
		draftPrompt := promptFiles[0]
		assert.Equal(t, "valid-draft", draftPrompt.Name)
		assert.Equal(t, string(entities.PromptTypeDrafts), draftPrompt.Type)
		assert.NotEmpty(t, draftPrompt.PromptTemplate)

		// Create entities and validate
		prompts, err := promptLoader.CreatePromptsFromFile("draft-user", promptFiles)
		require.NoError(t, err)
		require.Len(t, prompts, 1)

		// Validate that required variables exist
		draftEntity := prompts[0]
		assert.Contains(t, draftEntity.PromptTemplate, "{content}")
		assert.Contains(t, draftEntity.PromptTemplate, "{user_context}")

		// Should not contain ideas prompt variables
		assert.NotContains(t, draftEntity.PromptTemplate, "{name}")
		assert.NotContains(t, draftEntity.PromptTemplate, "{ideas}")
		assert.NotContains(t, draftEntity.PromptTemplate, "{[related_topics]}")
	})

	t.Run("should reject prompt templates missing required variables", func(t *testing.T) {
		// Create ideas prompt missing required {name} variable
		invalidFile := filepath.Join(tempDir, "invalid-ideas.md")
		invalidContent := `---
name: invalid-ideas
type: ideas
---
Eres un experto en crear contenido.
Genera {ideas} ideas.
Missing {name} variable.`

		err := os.WriteFile(invalidFile, []byte(invalidContent), 0644)
		require.NoError(t, err)

		// Load should succeed (template validation happens at processing time)
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)
		require.Len(t, promptFiles, 1)

		// Create entities
		prompts, err := promptLoader.CreatePromptsFromFile("invalid-user", promptFiles)
		require.NoError(t, err)
		require.Len(t, prompts, 1)

		// Store in database
		prompt := prompts[0]
		_, err = promptsRepo.Create(ctx, prompt)
		require.NoError(t, err)

		// Processing should fail with validation error
		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        "invalid-user",
			Name:          "", // Empty name will cause {name} to be empty
			Ideas:         3,
			RelatedTopics: []string{"Test"},
			Active:        true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		testUser := &entities.User{
			ID:    "invalid-user",
			Email: "invalid@example.com",
		}

		engine := NewPromptEngine(promptsRepo, logger)

		// Should fail validation when trying to process
		_, err = engine.ProcessPrompt(
			ctx,
			"invalid-user",
			"invalid-ideas",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			testUser,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required variable")
	})

	t.Run("should reject prompt templates with invalid type", func(t *testing.T) {
		// Create file with invalid type
		invalidTypeFile := filepath.Join(tempDir, "invalid-type.md")
		invalidTypeContent := `---
name: bad-type
type: invalid_prompt_type
---
This has an invalid type in front-matter.`

		err := os.WriteFile(invalidTypeFile, []byte(invalidTypeContent), 0644)
		require.NoError(t, err)

		// Load should skip invalid file
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)

		// Should not load the invalid file
		var foundInvalidPrompt *PromptFile
		for _, file := range promptFiles {
			if file.Name == "bad-type" {
				foundInvalidPrompt = file
				break
			}
		}
		assert.Nil(t, foundInvalidPrompt)
	})

	t.Run("should reject files with missing front-matter", func(t *testing.T) {
		// Create file without front-matter
		noFrontMatterFile := filepath.Join(tempDir, "no-frontmatter.md")
		noFrontMatterContent := `This file has no front-matter.
It shouldn't be loaded as a prompt.`

		err := os.WriteFile(noFrontMatterFile, []byte(noFrontMatterContent), 0644)
		require.NoError(t, err)

		// Load should skip file without valid front-matter
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)

		// Should not load the file
		var foundNoFrontMatter *PromptFile
		for _, file := range promptFiles {
			if file.Name == "" && file.Type == "" {
				foundNoFrontMatter = file
				break
			}
		}
		assert.Nil(t, foundNoFrontMatter)
	})

	t.Run("should reject files with missing required fields", func(t *testing.T) {
		// Create file missing name
		missingNameFile := filepath.Join(tempDir, "missing-name.md")
		missingNameContent := `---
type: ideas
---
Missing name field in front-matter.`

		err := os.WriteFile(missingNameFile, []byte(missingNameContent), 0644)
		require.NoError(t, err)

		// Load should skip file with missing name
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)

		// Should not load the file
		var foundMissingName *PromptFile
		for _, file := range promptFiles {
			if file.Type == "ideas" && file.Name == "" {
				foundMissingName = file
				break
			}
		}
		assert.Nil(t, foundMissingName)

		// Create file missing type
		missingTypeFile := filepath.Join(tempDir, "missing-type.md")
		missingTypeContent := `---
name: no-type
---
Missing type field in front-matter.`

		err = os.WriteFile(missingTypeFile, []byte(missingTypeContent), 0644)
		require.NoError(t, err)

		// Load should skip file with missing type
		promptFiles, err = promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)

		// Should not load the file
		var foundMissingType *PromptFile
		for _, file := range promptFiles {
			if file.Name == "no-type" && file.Type == "" {
				foundMissingType = file
				break
			}
		}
		assert.Nil(t, foundMissingType)
	})

	t.Run("should skip .old.md files during loading", func(t *testing.T) {
		// Create legacy file
		legacyFile := filepath.Join(tempDir, "legacy.idea.old.md")
		legacyContent := `---
name: legacy
type: ideas
---
This is an old file that should be ignored.`

		err := os.WriteFile(legacyFile, []byte(legacyContent), 0644)
		require.NoError(t, err)

		// Load should skip legacy files
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)

		// Should not load the legacy file
		var foundLegacy *PromptFile
		for _, file := range promptFiles {
			if file.Name == "legacy" {
				foundLegacy = file
				break
			}
		}
		assert.Nil(t, foundLegacy)
	})

	t.Run("should validate entity constraints", func(t *testing.T) {
		// Create file with valid content
		validFile := filepath.Join(tempDir, "entity-test.md")
		validContent := `---
name: entity-validator
type: ideas
---
Valid template with {name} and {ideas} variables.`

		err := os.WriteFile(validFile, []byte(validContent), 0644)
		require.NoError(t, err)

		// Create entities
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)

		prompts, err := promptLoader.CreatePromptsFromFile("entity-user", promptFiles)
		require.NoError(t, err)

		// Get the entity
		entity := prompts[0]

		// Should pass validation
		assert.NoError(t, entity.Validate())

		// Test invalid changes would fail validation
		originalName := entity.Name
		entity.Name = "" // Empty name should fail validation
		assert.Error(t, entity.Validate())

		entity.Name = originalName // Restore
		entity.PromptTemplate = "" // Empty template should fail
		assert.Error(t, entity.Validate())
	})

	t.Run("should handle yaml parsing errors", func(t *testing.T) {
		// Create file with invalid YAML
		invalidYamlFile := filepath.Join(tempDir, "invalid-yaml.md")
		invalidYamlContent := `---
name: yaml-test
type: ideas
invalid_yaml: [missing closing bracket
---
Template content after invalid YAML.`

		err := os.WriteFile(invalidYamlFile, []byte(invalidYamlContent), 0644)
		require.NoError(t, err)

		// Load should skip file with invalid YAML
		promptFiles, err := promptLoader.LoadPromptsFromDir(tempDir)
		require.NoError(t, err)

		// Should not load the file with invalid YAML
		var foundInvalidYaml *PromptFile
		for _, file := range promptFiles {
			if file.Type == "ideas" {
				foundInvalidYaml = file
				break
			}
		}
		assert.Nil(t, foundInvalidYaml)
	})
}

// TestPromptEngineValidation tests validation within PromptEngine
func TestPromptEngineValidation(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	testDB := setupTestDB(t, "test_engine_validation")
	defer testDB.Disconnect(ctx)

	promptsRepo := testDB.PromptsRepository()
	logger := &mockLogger{}
	engine := NewPromptEngine(promptsRepo, logger)

	t.Run("should validate required parameters based on prompt type", func(t *testing.T) {
		userID := "engine-validation-user"
		now := time.Now()

		testUser := &entities.User{
			ID:        userID,
			Email:     "engine@example.com",
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Test ideas prompt type requires topic
		_, err := engine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent",
			entities.PromptTypeIdeas,
			nil, // Missing required topic
			nil,
			testUser,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "topic is required")

		// Test drafts prompt type requires idea
		_, err = engine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent",
			entities.PromptTypeDrafts,
			nil,
			nil, // Missing required idea
			testUser,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "idea is required")

		// Test all prompts require user
		testTopic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Test",
			Ideas:         3,
			RelatedTopics: []string{"Test"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		_, err = engine.ProcessPrompt(
			ctx,
			userID,
			"nonexistent",
			entities.PromptTypeIdeas,
			testTopic,
			nil,
			nil, // Missing required user
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user is required")
	})
}

// Mock Logger
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
func (m *mockLogger) Info(msg string, fields ...interface{})  {}
func (m *mockLogger) Warn(msg string, fields ...interface{})  {}
func (m *mockLogger) Error(msg string, fields ...interface{}) {}
