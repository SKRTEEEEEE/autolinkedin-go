package services

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

// DevSeederMarkdownTestSuite contains tests for DevSeeder with markdown prompt parsing
type DevSeederMarkdownTestSuite struct {
	suite.Suite
	db     *database.Database
	seeder *DevSeeder
	ctx    context.Context
	cleanUp func()
}

// SetupSuite runs before all tests
func (suite *DevSeederMarkdownTestSuite) SetupSuite() {
	db, cleanUp := database.CreateTestDatabase(suite.T())
	suite.db = db
	suite.seeder = NewDevSeeder(db)
	suite.ctx = context.Background()
	suite.cleanUp = cleanUp
}

// TearDownSuite runs after all tests
func (suite *DevSeederMarkdownTestSuite) TearDownSuite() {
	suite.cleanUp()
}

// TestDevSeederMarkdownTestSuite runs the test suite
func TestDevSeederMarkdownTestSuite(t *testing.T) {
	suite.Run(t, new(DevSeederMarkdownTestSuite))
}

// Test_ParsePromptFromMarkdown_ShouldExtractYAMLFrontMatter tests markdown parsing
func (suite *DevSeederMarkdownTestSuite) Test_ParsePromptFromMarkdown_ShouldExtractYAMLFrontMatter() {
	// Given
	markdownContent := `---
name: base1
type: ideas
active: true
---
Generate 5 creative ideas about {topic_description} with focus on {keywords}

Make sure the ideas are:
- Original and engaging
- Relevant to the target audience
- Actionable and specific
`

	// When
	prompt, err := suite.seeder.ParsePromptFromMarkdown(markdownContent)

	// Then
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), prompt)
	assert.Equal(suite.T(), "base1", prompt.Name)
	assert.Equal(suite.T(), entities.PromptTypeIdeas, prompt.Type)
	assert.True(suite.T(), prompt.Active)
	assert.Contains(suite.T(), prompt.PromptTemplate, "Generate 5 creative ideas about {topic_description}")
}

// Test_ParsePromptFromMarkdown_ShouldHandleDraftPrompt tests draft prompt parsing
func (suite *DevSeederMarkdownTestSuite) Test_ParsePromptFromMarkdown_ShouldHandleDraftPrompt() {
	// Given
	markdownContent := `---
name: professional
type: drafts
active: true
---
Write a professional draft about {topic} using a formal tone.

Include:
- Clear introduction
- Well-structured body paragraphs
- Strong conclusion
- Professional language throughout
`

	// When
	prompt, err := suite.seeder.ParsePromptFromMarkdown(markdownContent)

	// Then
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), prompt)
	assert.Equal(suite.T(), "professional", prompt.Name)
	assert.Equal(suite.T(), entities.PromptTypeDrafts, prompt.Type)
	assert.True(suite.T(), prompt.Active)
	assert.Contains(suite.T(), prompt.PromptTemplate, "Write a professional draft about {topic}")
}

// Test_ParsePromptFromMarkdown_ShouldReturnErrorForInvalidYAML tests error handling
func (suite *DevSeederMarkdownTestSuite) Test_ParsePromptFromMarkdown_ShouldReturnErrorForInvalidYAML() {
	// Given
	invalidMarkdown := `---
name: base1
type: ideas
invalid: yaml: content: [unclosed
---
Generate prompt content here
`

	// When
	prompt, err := suite.seeder.ParsePromptFromMarkdown(invalidMarkdown)

	// Then
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), prompt)
	assert.Contains(suite.T(), err.Error(), "front matter")
}

// Test_ParsePromptFromMarkdown_ShouldUseDefaultValues tests default value handling
func (suite *DevSeederMarkdownTestSuite) Test_ParsePromptFromMarkdown_ShouldUseDefaultValues() {
	// Given
	minimalMarkdown := `---
name: minimal-prompt
type: ideas
---
Simple prompt template
`

	// When
	prompt, err := suite.seeder.ParsePromptFromMarkdown(minimalMarkdown)

	// Then
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), prompt)
	assert.Equal(suite.T(), "minimal-prompt", prompt.Name)
	assert.Equal(suite.T(), entities.PromptTypeIdeas, prompt.Type)
	assert.True(suite.T(), prompt.Active) // Should default to true
}

// Test_SeedPromptsFromMarkdown_ShouldReadFromSeedDirectory tests seeding from markdown files
func (suite *DevSeederMarkdownTestSuite) Test_SeedPromptsFromMarkdown_ShouldReadFromSeedDirectory() {
	// Given
	seedDir := "../../seed/prompts" // Relative to project root

	// When
	err := suite.seeder.SeedPromptsFromMarkdown(suite.ctx, "dev-user-id", seedDir)

	// Then
	require.NoError(suite.T(), err)
	
	// Verify that prompts were created
	prompts, err := suite.seeder.promptRepo.FindAllByUserID(suite.ctx, "dev-user-id")
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), len(prompts), 0)
	
	// Verify specific prompts exist
	base1Prompt, err := suite.seeder.promptRepo.FindByName(suite.ctx, "dev-user-id", "base1")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "base1", base1Prompt.Name)
}

// Test_SeedTopicsWithPromptReferences_ShouldSetPromptField tests topic seeding with prompt references
func (suite *DevSeederMarkdownTestSuite) Test_SeedTopicsWithPromptReferences_ShouldSetPromptField() {
	// Given
	// First seed prompts
	err := suite.seeder.SeedPromptsFromMarkdown(suite.ctx, "dev-user-id", "../../seed/prompts")
	require.NoError(suite.T(), err)

	// When
	err = suite.seeder.SeedTopicsWithPromptReferences(suite.ctx, "dev-user-id")

	// Then
	require.NoError(suite.T(), err)
	
	// Verify topics were created with prompt references
	topics, err := suite.seeder.topicRepo.FindAllByUserID(suite.ctx, "dev-user-id")
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), len(topics), 0)
	
	// Check that topics have the prompt field set
	for _, topic := range topics {
		assert.NotEmpty(suite.T(), topic.Prompt, "Topic should have prompt reference")
		
		// Verify the referenced prompt exists
		prompt, err := suite.seeder.promptRepo.FindByName(suite.ctx, "dev-user-id", topic.Prompt)
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), prompt)
	}
}

// Test_ValidatePromptReference_ShouldCheckPromptExists tests prompt validation
func (suite *DevSeederMarkdownTestSuite) Test_ValidatePromptReference_ShouldCheckPromptExists() {
	// Given
	userID := "dev-user-id"
	existingPrompt := &entities.Prompt{
		ID:             "prompt-123",
		UserID:         userID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Test template",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	err := suite.seeder.promptRepo.Create(suite.ctx, existingPrompt)
	require.NoError(suite.T(), err)

	// When/Then - Test existing prompt
	isValid := suite.seeder.ValidatePromptReference(suite.ctx, userID, "base1")
	assert.True(suite.T(), isValid)

	// When/Then - Test non-existing prompt
	isValid = suite.seeder.ValidatePromptReference(suite.ctx, userID, "non-existing")
	assert.False(suite.T(), isValid)
}
