package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestPromptRepositorySynchronization(t *testing.T) {
	// Test for the new synchronization methods in PromptRepository:
	// - SyncFromFilesystem: Sync seed files with database
	// - ValidateTemplate: Validate prompt template syntax
	// - SaveActiveState: Activate/deactivate prompts
	// - Template validation methods

	// Setup test database
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(context.Background())

	// Use a test database
	testDB := client.Database("test_prompts_sync")
	defer testDB.Drop(context.Background())

	// Create collection
	collection := testDB.Collection("prompts")
	repo := NewPromptsRepository(collection)

	// Clear collection before each test
	setup := func(t *testing.T) {
		_, err := collection.DeleteMany(context.Background(), bson.M{})
		require.NoError(t, err)
	}

	t.Run("should validate prompt template syntax correctly", func(t *testing.T) {
		// GIVEN a PromptRepository instance
		// WHEN validating templates

		// THEN valid templates should pass
		validTemplates := []string{
			"Genera {ideas} ideas sobre {name}",
			"Escribe sobre {content} con contexto: {user_context}",
			"Template sin variables válidas",
			"Multiple variables: {name}, {[related_topics]}, {ideas}",
		}

		for _, template := range validTemplates {
			err := repo.ValidateTemplate(template)
			assert.NoError(t, err, "Template should be valid: %s", template)
		}

		// AND invalid templates should fail
		invalidTemplates := []string{
			"", // Empty template
			"Template with {incomplete", // Unclosed variable
			"Template with invalid {unknown_variable}", // Unknown variable
			"Template with {{double} curly", // Malformed
		}

		for _, template := range invalidTemplates {
			err := repo.ValidateTemplate(template)
			assert.Error(t, err, "Template should be invalid: %s", template)
		}
	})

	t.Run("should identify all valid variables in templates", func(t *testing.T) {
		// GIVEN a PromptRepository instance
		// WHEN extracting variables from templates

		// THEN all variables should be identified correctly
		testCases := []struct {
			template    string
			expected    []string
			description string
		}{
			{
				template:    "Genera {ideas} ideas sobre {name}",
				expected:    []string{"ideas", "name"},
				description: "Standard variables",
			},
			{
				template:    "Topics: {[related_topics]} and content: {content}",
				expected:    []string{ "[related_topics]", "content"},
				description: "Array and content variable",
			},
			{
				template:    "No variables here",
				expected:    []string{},
				description: "Template without variables",
			},
			{
				template:    "Repeated: {name} and again {name}",
				expected:    []string{"name"},
				description: "Duplicated variables should be unique",
			},
		}

		for _, tc := range testCases {
			variables, err := repo.ExtractVariables(tc.template)
			require.NoError(t, err, tc.description)
			assert.Equal(t, tc.expected, variables, tc.description)
		}
	})

	t.Run("should check if all required variables are present", func(t *testing.T) {
		// GIVEN a PromptRepository instance
		// WHEN checking variables by prompt type

		// THEN validation should work correctly
		typeTestCases := []struct {
			promptType entities.PromptType
			template   string
			shouldPass bool
			description string
		}{
			{
				promptType: entities.PromptTypeIdeas,
				template:   "Genera {ideas} ideas sobre {name}",
				shouldPass: true,
				description: "Valid ideas prompt template",
			},
			{
				promptType: entities.PromptTypeIdeas,
				template:   "Genera ideas sobre {name}", // Missing {ideas}
				shouldPass: false,
				description: "Ideas prompt missing required {ideas}",
			},
			{
				promptType: entities.PromptTypeIdeas,
				template:   "Genera {ideas} ideas", // Missing {name}
				shouldPass: false,
				description: "Ideas prompt missing required {name}",
			},
			{
				promptType: entities.PromptTypeDrafts,
				template:   "Escribe sobre {content} con contexto {user_context}",
				shouldPass: true,
				description: "Valid drafts prompt template",
			},
			{
				promptType: entities.PromptTypeDrafts,
				template:   "Escribe sobre {content}", // Missing {user_context}
				shouldPass: false,
				description: "Drafts prompt missing required {user_context}",
			},
		}

		for _, tc := range typeTestCases {
			err := repo.ValidateVariablesByType(tc.template, tc.promptType)
			if tc.shouldPass {
				assert.NoError(t, err, tc.description)
			} else {
				assert.Error(t, err, tc.description)
			}
		}
	})

	t.Run("should activate and deactivate prompts by name", func(t *testing.T) {
		setup(t)

		// GIVEN a PromptRepository with existing prompts
		userID := "user-123"
		promptName := "test-prompt"

		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           promptName,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Template with {name}",
			Active:         false,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Insert inactive prompt
		id, err := repo.Create(context.Background(), prompt)
		require.NoError(t, err)
		prompt.ID = id

		// WHEN activating the prompt
		err = repo.SetActiveStatus(context.Background(), userID, promptName, true)
		require.NoError(t, err)

		// THEN the prompt should be active
		activePrompt, err := repo.FindActiveByName(context.Background(), userID, promptName)
		require.NoError(t, err)
		assert.NotNil(t, activePrompt)
		assert.True(t, activePrompt.Active)

		// WHEN deactivating the prompt
		err = repo.SetActiveStatus(context.Background(), userID, promptName, false)
		require.NoError(t, err)

		// THEN the prompt should be inactive
		inactiveResult, err := repo.FindActiveByName(context.Background(), userID, promptName)
		require.NoError(t, err)
		assert.Nil(t, inactiveResult)

		// Verify it still exists but is inactive
		stillExists, err := repo.FindByName(context.Background(), userID, promptName)
		require.NoError(t, err)
		assert.NotNil(t, stillExists)
		assert.False(t, stillExists.Active)
	})

	t.Run("should find prompts by name with type validation", func(t *testing.T) {
		setup(t)

		// GIVEN prompts with different types but same name
		userID := "user-456"
		promptName := "dual-type"

		ideasPrompt := &entities.Prompt{
			Name:           promptName,
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Ideas template with {ideas}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		draftsPrompt := &entities.Prompt{
			Name:           promptName,
			UserID:         userID,
			Type:           entities.PromptTypeDrafts,
			PromptTemplate: "Drafts template with {content}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Insert both prompts
		_, err := repo.Create(context.Background(), ideasPrompt)
		require.NoError(t, err)
		_, err = repo.Create(context.Background(), draftsPrompt)
		require.NoError(t, err)

		// WHEN finding by name and type
		foundIdeas, err := repo.FindByNameAndType(context.Background(), userID, promptName, entities.PromptTypeIdeas)
		require.NoError(t, err)
		assert.NotNil(t, foundIdeas)
		assert.Equal(t, entities.PromptTypeIdeas, foundIdeas.Type)
		assert.Contains(t, foundIdeas.PromptTemplate, "Ideas template")

		foundDrafts, err := repo.FindByNameAndType(context.Background(), userID, promptName, entities.PromptTypeDrafts)
		require.NoError(t, err)
		assert.NotNil(t, foundDrafts)
		assert.Equal(t, entities.PromptTypeDrafts, foundDrafts.Type)
		assert.Contains(t, foundDrafts.PromptTemplate, "Drafts template")
	})

	t.Run("should reset user prompts to default from file system", func(t *testing.T) {
		setup(t)

		// GIVEN a user with custom prompts
		userID := "user-789"
		seedDir := "../../seed/prompt" // Relative to project root

		// Create custom prompts
		customPrompt1 := &entities.Prompt{
			Name:           "base1",
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Custom ideas template",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		customPrompt2 := &entities.Prompt{
			Name:           "pro",
			UserID:         userID,
			Type:           entities.PromptTypeDrafts,
			PromptTemplate: "Custom drafts template",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Insert custom prompts
		_, err := repo.Create(context.Background(), customPrompt1)
		require.NoError(t, err)
		_, err = repo.Create(context.Background(), customPrompt2)
		require.NoError(t, err)

		// WHEN resetting to defaults
		err = repo.ResetToDefaults(context.Background(), userID, seedDir)
		require.NoError(t, err)

		// THEN prompts should be reset to seed defaults
		// Get all prompts for user
		userPrompts, err := repo.ListByUserID(context.Background(), userID)
		require.NoError(t, err)
		assert.Greater(t, len(userPrompts), 0)

		// Verify base1 was reset
		base1Prompt, err := repo.FindByNameAndType(context.Background(), userID, "base1", entities.PromptTypeIdeas)
		require.NoError(t, err)
		assert.NotNil(t, base1Prompt)
		assert.Contains(t, base1Prompt.PromptTemplate, "Genera {ideas} ideas") // Content from seed file
		assert.False(t, base1Prompt.Active) // Should be inactive until explicitly activated

		// Verify pro was reset
		proPrompt, err := repo.FindByNameAndType(context.Background(), userID, "pro", entities.PromptTypeDrafts)
		require.NoError(t, err)
		assert.NotNil(t, proPrompt)
		assert.Contains(t, proPrompt.PromptTemplate, "Escribe un post profesional") // Content from seed file
		assert.False(t, proPrompt.Active) // Should be inactive until explicitly activated
	})

	t.Run("should validate variable placeholders format", func(t *testing.T) {
		// GIVEN a PromptRepository instance
		// WHEN validating variable format

		// THEN formats should be validated correctly
		validFormats := []string{
			"{name}",
			"{ideas}",
			"{content}",
			"{user_context}",
			"{[related_topics]}", // Array-style variable
		}

		for _, variable := range validFormats {
			err := repo.ValidateVariableFormat(variable)
			assert.NoError(t, err, "Variable format should be valid: %s", variable)
		}

		invalidFormats := []string{
			"{name", // Missing closing brace
			"name}", // Missing opening brace
			"{name}", // Extra space
			"{name-variable}", // Hyphen not allowed
			"{123name}", // Numbers not allowed at start
			"{name!}", // Exclamation not allowed
		}

		for _, variable := range invalidFormats {
			err := repo.ValidateVariableFormat(variable)
			assert.Error(t, err, "Variable format should be invalid: %s", variable)
		}
	})
}

// NEW methods to be added to PromptsRepository

// ValidateTemplate validates the syntax of a prompt template
func (r *PromptsRepository) ValidateTemplate(template string) error {
	// TODO: Implementation needed - this test will fail until implemented
	return assert.AnError
}

// ExtractVariables extracts all variable placeholders from a template
func (r *PromptsRepository) ExtractVariables(template string) ([]string, error) {
	// TODO: Implementation needed - this test will fail until implemented
	return nil, assert.AnError
}

// ValidateVariablesByType validates that all required variables for a prompt type are present
func (r *PromptsRepository) ValidateVariablesByType(template string, promptType entities.PromptType) error {
	// TODO: Implementation needed - this test will fail until implemented
	return assert.AnError
}

// SetActiveStatus activates or deactivates a prompt by name and user
func (r *PromptsRepository) SetActiveStatus(ctx context.Context, userID string, name string, active bool) error {
	// TODO: Implementation needed - this test will fail until implemented
	return assert.AnError
}

// FindByNameAndType finds a prompt by name, user ID, and type
func (r *PromptsRepository) FindByNameAndType(ctx context.Context, userID string, name string, promptType entities.PromptType) (*entities.Prompt, error) {
	// TODO: Implementation needed - this test will fail until implemented
	return nil, assert.AnError
}

// ResetToDefaults resets user prompts to default values from seed files
func (r *PromptsRepository) ResetToDefaults(ctx context.Context, userID string, seedDirectory string) error {
	// TODO: Implementation needed - this test will fail until implemented
	return assert.AnError
}

// ValidateVariableFormat validates the format of a single variable
func (r *PromptsRepository) ValidateVariableFormat(variable string) error {
	// TODO: Implementation needed - this test will fail until implemented
	return assert.AnError
}
