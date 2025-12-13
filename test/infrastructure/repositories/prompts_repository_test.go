package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PromptsRepositoryTestSuite contains all tests for PromptsRepository
type PromptsRepositoryTestSuite struct {
	suite.Suite
	db      *database.Database
	repo    *PromptsRepository
	ctx     context.Context
	cleanUp func()
}

// SetupSuite runs before all tests
func (suite *PromptsRepositoryTestSuite) SetupSuite() {
	db, cleanUp := database.CreateTestDatabase(suite.T())
	suite.db = db
	suite.repo = NewPromptsRepository(db)
	suite.ctx = context.Background()
	suite.cleanUp = cleanUp
}

// TearDownSuite runs after all tests
func (suite *PromptsRepositoryTestSuite) TearDownSuite() {
	suite.cleanUp()
}

// TestPromptsRepositoryTestSuite runs the test suite
func TestPromptsRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(PromptsRepositoryTestSuite))
}

// Test_FindByName_ShouldReturnPromptByName tests the new FindByName method
func (suite *PromptsRepositoryTestSuite) Test_FindByName_ShouldReturnPromptByName() {
	// Given
	userID := "user-123"
	expectedPrompt := &entities.Prompt{
		ID:             "prompt-123",
		UserID:         userID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate ideas about {topic}",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := suite.repo.Create(suite.ctx, expectedPrompt)
	require.NoError(suite.T(), err)

	// When
	foundPrompt, err := suite.repo.FindByName(suite.ctx, userID, "base1")

	// Then
	assert.NoError(suite.T(), err)
	require.NotNil(suite.T(), foundPrompt)
	assert.Equal(suite.T(), expectedPrompt.ID, foundPrompt.ID)
	assert.Equal(suite.T(), expectedPrompt.Name, foundPrompt.Name)
}

// Test_FindByName_ShouldReturnErrorForNonExistingName tests error case
func (suite *PromptsRepositoryTestSuite) Test_FindByName_ShouldReturnErrorForNonExistingName() {
	// Given
	userID := "user-123"
	nonExistingName := "non-existing-prompt"

	// When
	foundPrompt, err := suite.repo.FindByName(suite.ctx, userID, nonExistingName)

	// Then
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), foundPrompt)
	assert.Contains(suite.T(), err.Error(), "prompt not found")
}

// Test_FindActiveByName_ShouldOnlyReturnActivePrompts tests filtering by active status
func (suite *PromptsRepositoryTestSuite) Test_FindActiveByName_ShouldOnlyReturnActivePrompts() {
	// Given
	userID := "user-123"

	// Create active prompt
	activePrompt := &entities.Prompt{
		ID:             "prompt-active",
		UserID:         userID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate ideas about {topic}",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create inactive prompt with same name
	inactivePrompt := &entities.Prompt{
		ID:             "prompt-inactive",
		UserID:         userID,
		Name:           "base2",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate ideas about {topic}",
		Active:         false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := suite.repo.Create(suite.ctx, activePrompt)
	require.NoError(suite.T(), err)
	err = suite.repo.Create(suite.ctx, inactivePrompt)
	require.NoError(suite.T(), err)

	// When
	foundPrompt, err := suite.repo.FindActiveByName(suite.ctx, userID, "base2")

	// Then
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), foundPrompt)
	assert.Contains(suite.T(), err.Error(), "active prompt not found")
}

// Test_Create_ShouldHandleNewNameField tests Create with new Name field
func (suite *PromptsRepositoryTestSuite) Test_Create_ShouldHandleNewNameField() {
	// Given
	prompt := &entities.Prompt{
		ID:             "prompt-new",
		UserID:         "user-123",
		Name:           "custom-name",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Custom template with new {name} field",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// When
	err := suite.repo.Create(suite.ctx, prompt)

	// Then
	assert.NoError(suite.T(), err)

	// Verify the prompt was created with the Name field
	found, err := suite.repo.FindByID(suite.ctx, prompt.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), prompt.Name, found.Name)
}

// Test_Update_ShouldHandleNameField tests Update with new Name field
func (suite *PromptsRepositoryTestSuite) Test_Update_ShouldHandleNameField() {
	// Given
	originalPrompt := &entities.Prompt{
		ID:             "prompt-update",
		UserID:         "user-123",
		Name:           "original-name",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Original template",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := suite.repo.Create(suite.ctx, originalPrompt)
	require.NoError(suite.T(), err)

	// Modify the prompt
	originalPrompt.Name = "updated-name"
	originalPrompt.PromptTemplate = "Updated template"
	originalPrompt.UpdatedAt = time.Now()

	// When
	err = suite.repo.Update(suite.ctx, originalPrompt)

	// Then
	assert.NoError(suite.T(), err)

	// Verify the update
	updated, err := suite.repo.FindByID(suite.ctx, originalPrompt.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "updated-name", updated.Name)
	assert.Equal(suite.T(), "Updated template", updated.PromptTemplate)
}
