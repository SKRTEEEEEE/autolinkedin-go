package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database"
	"github.com/linkgen-ai/backend/src/infrastructure/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// GenerateIdeasUseCasePromptRefTestSuite contains tests for GenerateIdeasUseCase with prompt references
type GenerateIdeasUseCasePromptRefTestSuite struct {
	suite.Suite
	db         *database.Database
	topicRepo  *repositories.TopicsRepository
	ideaRepo   *repositories.IdeasRepository
	promptRepo *repositories.PromptsRepository
	useCase    *GenerateIdeasUseCase
	ctx        context.Context
	cleanUp    func()
}

// SetupSuite runs before all tests
func (suite *GenerateIdeasUseCasePromptRefTestSuite) SetupSuite() {
	db, cleanUp := database.CreateTestDatabase(suite.T())
	suite.db = db
	suite.topicRepo = repositories.NewTopicsRepository(db)
	suite.ideaRepo = repositories.NewIdeasRepository(db)
	suite.promptRepo = repositories.NewPromptsRepository(db)
	suite.useCase = NewGenerateIdeasUseCase(suite.topicRepo, suite.ideaRepo, suite.promptRepo)
	suite.ctx = context.Background()
	suite.cleanUp = cleanUp
}

// TearDownSuite runs after all tests
func (suite *GenerateIdeasUseCasePromptRefTestSuite) TearDownSuite() {
	suite.cleanUp()
}

// TestGenerateIdeasUseCasePromptRefTestSuite runs the test suite
func TestGenerateIdeasUseCasePromptRefTestSuite(t *testing.T) {
	suite.Run(t, new(GenerateIdeasUseCasePromptRefTestSuite))
}

// Test_Execute_ShouldUseTopicSpecificPrompt tests using topic's prompt reference
func (suite *GenerateIdeasUseCasePromptRefTestSuite) Test_Execute_ShouldUseTopicSpecificPrompt() {
	// Given
	userID := "user-123"

	// Create a custom prompt
	customPrompt := &entities.Prompt{
		ID:             "prompt-custom",
		UserID:         userID,
		Name:           "creative-style",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate {{ideas}} creative ideas about {topic_description} with focus on {keywords}. Make them innovative and thought-provoking.",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := suite.promptRepo.Create(suite.ctx, customPrompt)
	require.NoError(suite.T(), err)

	// Create topic with custom prompt reference
	topic := &entities.Topic{
		ID:          "topic-123",
		UserID:      userID,
		Name:        "AI Innovation",
		Description: "Exploring AI innovation",
		Keywords:    []string{"AI", "Innovation", "Technology"},
		Category:    "Technology",
		Priority:    8,
		Ideas:       3,
		Prompt:      "creative-style", // Using custom prompt
		Active:      true,
		CreatedAt:   time.Now(),
	}
	err = suite.topicRepo.Create(suite.ctx, topic)
	require.NoError(suite.T(), err)

	// When
	ideas, err := suite.useCase.Execute(suite.ctx, userID, topic.ID)

	// Then
	require.NoError(suite.T(), err)
	require.Len(suite.T(), ideas, 3)

	// Verify the generated ideas belong to the topic and have topic_name
	for _, idea := range ideas {
		assert.Equal(suite.T(), userID, idea.UserID)
		assert.Equal(suite.T(), topic.ID, idea.TopicID)
		assert.Equal(suite.T(), topic.Name, idea.TopicName) // Should match topic name
		assert.NotEmpty(suite.T(), idea.Content)
	}
}

// Test_Execute_ShouldReturnErrorForNonExistingPrompt tests error when referenced prompt doesn't exist
func (suite *GenerateIdeasUseCasePromptRefTestSuite) Test_Execute_ShouldReturnErrorForNonExistingPrompt() {
	// Given
	userID := "user-123"

	// Create topic with non-existing prompt reference
	topic := &entities.Topic{
		ID:          "topic-123",
		UserID:      userID,
		Name:        "Test Topic",
		Description: "Test description",
		Keywords:    []string{"test"},
		Category:    "Test",
		Priority:    5,
		Ideas:       2,
		Prompt:      "non-existing-prompt", // This prompt doesn't exist
		Active:      true,
		CreatedAt:   time.Now(),
	}
	err := suite.topicRepo.Create(suite.ctx, topic)
	require.NoError(suite.T(), err)

	// When
	ideas, err := suite.useCase.Execute(suite.ctx, userID, topic.ID)

	// Then
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), ideas)
	assert.Contains(suite.T(), err.Error(), "prompt reference not found")
}

// Test_Execute_ShouldUseDefaultPromptWhenNotSpecified tests fallback to default prompt
func (suite *GenerateIdeasUseCasePromptRefTestSuite) Test_Execute_ShouldUseDefaultPromptWhenNotSpecified() {
	// Given
	userID := "user-123"

	// Create the default base1 prompt
	basePrompt := &entities.Prompt{
		ID:             "prompt-base",
		UserID:         userID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate {{ideas}} ideas about {topic_description} with focus on {keywords}",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := suite.promptRepo.Create(suite.ctx, basePrompt)
	require.NoError(suite.T(), err)

	// Create topic without prompt reference
	topic := &entities.Topic{
		ID:          "topic-123",
		UserID:      userID,
		Name:        "Test Topic",
		Description: "Test description",
		Keywords:    []string{"test"},
		Category:    "Test",
		Priority:    5,
		Ideas:       2,
		Prompt:      "", // Empty prompt field
		Active:      true,
		CreatedAt:   time.Now(),
	}
	err = suite.topicRepo.Create(suite.ctx, topic)
	require.NoError(suite.T(), err)

	// When
	ideas, err := suite.useCase.Execute(suite.ctx, userID, topic.ID)

	// Then
	require.NoError(suite.T(), err)
	require.Len(suite.T(), ideas, 2)

	// Should still generate ideas with topic_name set
	for _, idea := range ideas {
		assert.Equal(suite.T(), topic.Name, idea.TopicName)
	}
}

// Test_Execute_ShouldUseIdeasCountFromTopic tests using the Ideas field from topic
func (suite *GenerateIdeasUseCasePromptRefTestSuite) Test_Execute_ShouldUseIdeasCountFromTopic() {
	// Given
	userID := "user-123"

	// Create prompt
	prompt := &entities.Prompt{
		ID:             "prompt-test",
		UserID:         userID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate {{ideas}} ideas about {topic_description}",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := suite.promptRepo.Create(suite.ctx, prompt)
	require.NoError(suite.T(), err)

	// Create topic with custom ideas count
	topic := &entities.Topic{
		ID:          "topic-123",
		UserID:      userID,
		Name:        "Test Topic",
		Description: "Test description",
		Keywords:    []string{"test"},
		Category:    "Test",
		Priority:    5,
		Ideas:       7, // Generate 7 ideas
		Prompt:      "base1",
		Active:      true,
		CreatedAt:   time.Now(),
	}
	err = suite.topicRepo.Create(suite.ctx, topic)
	require.NoError(suite.T(), err)

	// When
	ideas, err := suite.useCase.Execute(suite.ctx, userID, topic.ID)

	// Then
	require.NoError(suite.T(), err)
	require.Len(suite.T(), ideas, 7) // Should generate exactly 7 ideas as specified
}

// Test_Execute_ShouldSetTopicNameOnGeneratedIdeas tests topic_name field is properly set
func (suite *GenerateIdeasUseCasePromptRefTestSuite) Test_Execute_ShouldSetTopicNameOnGeneratedIdeas() {
	// Given
	userID := "user-123"

	// Create prompt
	prompt := &entities.Prompt{
		ID:             "prompt-test",
		UserID:         userID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate {{ideas}} ideas about {topic_description}",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := suite.promptRepo.Create(suite.ctx, prompt)
	require.NoError(suite.T(), err)

	// Create topic
	topic := &entities.Topic{
		ID:          "topic-123",
		UserID:      userID,
		Name:        "Machine Learning Research",
		Description: "Latest ML research topics",
		Keywords:    []string{"ML", "Research", "AI"},
		Category:    "Technology",
		Priority:    8,
		Ideas:       3,
		Prompt:      "base1",
		Active:      true,
		CreatedAt:   time.Now(),
	}
	err = suite.topicRepo.Create(suite.ctx, topic)
	require.NoError(suite.T(), err)

	// When
	ideas, err := suite.useCase.Execute(suite.ctx, userID, topic.ID)

	// Then
	require.NoError(suite.T(), err)
	require.Len(suite.T(), ideas, 3)

	// Verify all ideas have the correct topic_name
	for _, idea := range ideas {
		assert.Equal(suite.T(), "Machine Learning Research", idea.TopicName)
		assert.NotEmpty(suite.T(), idea.Content)

		// Verify the idea is a valid entity
		err = idea.Validate()
		assert.NoError(suite.T(), err)
	}
}
