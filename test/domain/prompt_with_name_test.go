package domain

import (
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewPrompt_ShouldCreatePromptWithNameField(t *testing.T) {
	// Given
	id := "prompt-123"
	userID := "user-456"
	name := "base1" // New Name field instead of StyleName
	promptType := entities.PromptTypeIdeas
	template := "Generate 5 ideas about {topic_description} focused on {keywords}"

	// When
	prompt, err := entities.NewPrompt(id, userID, name, promptType, template)

	// Then
	require.NoError(t, err)
	assert.Equal(t, id, prompt.ID)
	assert.Equal(t, userID, prompt.UserID)
	assert.Equal(t, name, prompt.Name) // Name field should be set
	assert.Equal(t, promptType, prompt.Type)
	assert.Equal(t, template, prompt.PromptTemplate)
	assert.True(t, prompt.Active) // Default should be active
}

func Test_NewPromptForDrafts_ShouldCreatePromptWithName(t *testing.T) {
	// Given
	id := "prompt-456"
	userID := "user-789"
	name := "professional"
	promptType := entities.PromptTypeDrafts
	template := "Write a professional draft about {topic} using tone: {tone}"

	// When
	prompt, err := entities.NewPrompt(id, userID, name, promptType, template)

	// Then
	require.NoError(t, err)
	assert.Equal(t, name, prompt.Name)
	assert.Equal(t, promptType, prompt.Type)
	assert.Equal(t, template, prompt.PromptTemplate)
}

func Test_Prompt_Validate_ShouldEnforceNameField(t *testing.T) {
	// Given
	prompt := &entities.Prompt{
		ID:             "prompt-123",
		UserID:         "user-456",
		Name:           "", // Invalid: empty name
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Valid template with enough length",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// When
	err := prompt.Validate()

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func Test_Prompt_Validate_ShouldEnforceNameLength(t *testing.T) {
	// Given
	longName := "This is a very long prompt name that exceeds maximum length limit"
	prompt := &entities.Prompt{
		ID:             "prompt-123",
		UserID:         "user-456",
		Name:           longName,
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Valid template with enough length",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// When
	err := prompt.Validate()

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name too long")
}

func Test_Prompt_Validate_ShouldRequireNameForDraftPrompts(t *testing.T) {
	// Given
	prompt := &entities.Prompt{
		ID:             "prompt-123",
		UserID:         "user-456",
		Name:           "", // Name should be required for draft prompts
		Type:           entities.PromptTypeDrafts,
		PromptTemplate: "Valid template with enough length for draft generation",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// When
	err := prompt.Validate()

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required for draft prompts")
}

func Test_Prompt_SetName_ShouldUpdateNameField(t *testing.T) {
	// Given
	prompt, _ := entities.NewPrompt("prompt-123", "user-456", "old-name", entities.PromptTypeIdeas, "Template")
	newName := "new-name"

	// When
	err := prompt.SetName(newName)

	// Then
	require.NoError(t, err)
	assert.Equal(t, newName, prompt.Name)
	assert.True(t, prompt.UpdatedAt.After(prompt.CreatedAt))
}

func Test_Prompt_ByNameShouldeferToNameField(t *testing.T) {
	// Given
	name := "creative-writing"
	prompt, _ := entities.NewPrompt("prompt-123", "user-456", name, entities.PromptTypeDrafts, "Template")

	// When
	retrievedName := prompt.GetName()

	// Then
	assert.Equal(t, name, retrievedName)
}
