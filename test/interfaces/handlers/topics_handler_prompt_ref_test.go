package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database"
	"github.com/linkgen-ai/backend/src/infrastructure/repositories"
	"github.com/linkgen-ai/backend/src/interfaces/handlers/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TopicsHandlerPromptRefTestSuite contains tests for TopicsHandler with prompt references
type TopicsHandlerPromptRefTestSuite struct {
	suite.Suite
	db         *database.Database
	topicRepo  *repositories.TopicsRepository
	promptRepo *repositories.PromptsRepository
	handler    *TopicsHandler
	router     *mux.Router
	cleanUp    func()
}

// SetupSuite runs before all tests
func (suite *TopicsHandlerPromptRefTestSuite) SetupSuite() {
	db, cleanUp := database.CreateTestDatabase(suite.T())
	suite.db = db
	suite.topicRepo = repositories.NewTopicsRepository(db)
	suite.promptRepo = repositories.NewPromptsRepository(db)
	suite.handler = NewTopicsHandler(suite.topicRepo, suite.promptRepo)
	suite.router = mux.NewRouter()
	suite.cleanUp = cleanUp
}

// TearDownSuite runs after all tests
func (suite *TopicsHandlerPromptRefTestSuite) TearDownSuite() {
	suite.cleanUp()
}

// TestTopicsHandlerPromptRefTestSuite runs the test suite
func TestTopicsHandlerPromptRefTestSuite(t *testing.T) {
	suite.Run(t, new(TopicsHandlerPromptRefTestSuite))
}

// Test_CreateTopic_ShouldHandlePromptReference tests creating topic with prompt field
func (suite *TopicsHandlerPromptRefTestSuite) Test_CreateTopic_ShouldHandlePromptReference() {
	// Given
	userID := "user-123"
	
	// Create a prompt first
	prompt := &entities.Prompt{
		ID:             "prompt-123",
		UserID:         userID,
		Name:           "custom-style",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Custom template",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := suite.promptRepo.Create(context.Background(), prompt)
	require.NoError(suite.T(), err)

	// Prepare request with prompt reference
	createTopicDTO := dto.CreateTopicDTO{
		Name:        "AI Innovation",
		Description: "Exploring AI innovations",
		Keywords:    []string{"AI", "Innovation"},
		Category:    "Technology",
		Priority:    8,
		Ideas:       5,
		Prompt:      "custom-style", // New field
	}

	body, _ := json.Marshal(createTopicDTO)
	req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	w := httptest.NewRecorder()

	// When
	suite.router.HandleFunc("/topics", suite.handler.Create).Methods("POST")
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusCreated, w.Code)
	
	var response dto.TopicDTO
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), createTopicDTO.Name, response.Name)
	assert.Equal(suite.T(), createTopicDTO.Prompt, response.Prompt)
	assert.Equal(suite.T(), createTopicDTO.Ideas, response.Ideas)
	
	// Verify topic was created in database with prompt reference
	topic, err := suite.topicRepo.FindByID(context.Background(), response.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "custom-style", topic.Prompt)
	assert.Equal(suite.T(), 5, topic.Ideas)
}

// Test_CreateTopic_ShouldReturnErrorForInvalidPromptReference tests validation of prompt field
func (suite *TopicsHandlerPromptRefTestSuite) Test_CreateTopic_ShouldReturnErrorForInvalidPromptReference() {
	// Given
	userID := "user-123"
	
	// Prepare request with non-existing prompt reference
	createTopicDTO := dto.CreateTopicDTO{
		Name:        "Test Topic",
		Description: "Test description",
		Keywords:    []string{"test"},
		Category:    "Test",
		Priority:    5,
		Ideas:       2,
		Prompt:      "non-existing-prompt", // Doesn't exist
	}

	body, _ := json.Marshal(createTopicDTO)
	req := httptest.NewRequest("POST", "/topics", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	w := httptest.NewRecorder()

	// When
	suite.router.HandleFunc("/topics", suite.handler.Create).Methods("POST")
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	
	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), errorResponse["message"], "prompt reference not found")
}

// Test_UpdateTopic_ShouldUpdatePromptReference tests updating topic's prompt field
func (suite *TopicsHandlerPromptRefTestSuite) Test_UpdateTopic_ShouldUpdatePromptReference() {
	// Given
	userID := "user-123"
	
	// Create two prompts
	prompt1 := &entities.Prompt{
		ID:             "prompt-1",
		UserID:         userID,
		Name:           "style1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Template 1",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	prompt2 := &entities.Prompt{
		ID:             "prompt-2",
		UserID:         userID,
		Name:           "style2",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Template 2",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	err := suite.promptRepo.Create(context.Background(), prompt1)
	require.NoError(suite.T(), err)
	err = suite.promptRepo.Create(context.Background(), prompt2)
	require.NoError(suite.T(), err)
	
	// Create topic with initial prompt
	topic := &entities.Topic{
		ID:          "topic-123",
		UserID:      userID,
		Name:        "Test Topic",
		Description: "Test description",
		Keywords:    []string{"test"},
		Category:    "Test",
		Priority:    5,
		Ideas:       2,
		Prompt:      "style1",
		Active:      true,
		CreatedAt:   time.Now(),
	}
	err = suite.topicRepo.Create(context.Background(), topic)
	require.NoError(suite.T(), err)

	// Prepare update request - changing prompt and ideas
	updateTopicDTO := dto.UpdateTopicDTO{
		Name:   &([]string{"Updated Topic"})[0],
		Prompt: &([]string{"style2"})[0], // Change prompt
		Ideas:  &([]int{5})[0],           // Update ideas count
	}

	body, _ := json.Marshal(updateTopicDTO)
	req := httptest.NewRequest("PUT", "/topics/"+topic.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	w := httptest.NewRecorder()

	// When
	suite.router.HandleFunc("/topics/{id}", suite.handler.Update).Methods("PUT")
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response dto.TopicDTO
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "style2", response.Prompt)
	assert.Equal(suite.T(), 5, response.Ideas)
	
	// Verify database was updated
	updatedTopic, err := suite.topicRepo.FindByID(context.Background(), topic.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "style2", updatedTopic.Prompt)
	assert.Equal(suite.T(), 5, updatedTopic.Ideas)
}

// Test_GetTopic_ShouldIncludePromptAndIdeasField tests topic response includes new fields
func (suite *TopicsHandlerPromptRefTestSuite) Test_GetTopic_ShouldIncludePromptAndIdeasField() {
	// Given
	userID := "user-123"
	
	// Create topic with new fields
	topic := &entities.Topic{
		ID:          "topic-123",
		UserID:      userID,
		Name:        "Test Topic",
		Description: "Test description",
		Keywords:    []string{"test"},
		Category:    "Test",
		Priority:    5,
		Ideas:       7,
		Prompt:      "base1",
		Active:      true,
		CreatedAt:   time.Now(),
	}
	err := suite.topicRepo.Create(context.Background(), topic)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest("GET", "/topics/"+topic.ID, nil)
	req.Header.Set("X-User-ID", userID)
	w := httptest.NewRecorder()

	// When
	suite.router.HandleFunc("/topics/{id}", suite.handler.GetByID).Methods("GET")
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response dto.TopicDTO
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), topic.ID, response.ID)
	assert.Equal(suite.T(), topic.Name, response.Name)
	assert.Equal(suite.T(), topic.Prompt, response.Prompt)
	assert.Equal(suite.T(), topic.Ideas, response.Ideas)
}

// Test_ListTopics_ShouldIncludePromptAndIdeasField tests topic list includes new fields
func (suite *TopicsHandlerPromptRefTestSuite) Test_ListTopics_ShouldIncludePromptAndIdeasField() {
	// Given
	userID := "user-123"
	
	// Create multiple topics with different prompts/ideas
	topics := []*entities.Topic{
		{
			ID:       "topic-1",
			UserID:   userID,
			Name:     "Topic 1",
			Ideas:    3,
			Prompt:   "base1",
			Active:   true,
			Priority: 5,
			CreatedAt: time.Now(),
		},
		{
			ID:       "topic-2",
			UserID:   userID,
			Name:     "Topic 2",
			Ideas:    5,
			Prompt:   "creative",
			Active:   true,
			Priority: 8,
			CreatedAt: time.Now(),
		},
	}
	
	for _, topic := range topics {
		err := suite.topicRepo.Create(context.Background(), topic)
		require.NoError(suite.T(), err)
	}

	req := httptest.NewRequest("GET", "/topics", nil)
	req.Header.Set("X-User-ID", userID)
	w := httptest.NewRecorder()

	// When
	suite.router.HandleFunc("/topics", suite.handler.List).Methods("GET")
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response dto.ListTopicsDTO
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), response.Topics, 2)
	
	// Verify topics include new fields
	for _, topicDTO := range response.Topics {
		assert.NotEmpty(suite.T(), topicDTO.Prompt)
		assert.Greater(suite.T(), topicDTO.Ideas, 0)
	}
}

// Test_ValidatePromptReference_ShouldCheckIfPromptExists tests prompt validation
func (suite *TopicsHandlerPromptRefTestSuite) Test_ValidatePromptReference_ShouldCheckIfPromptExists() {
	// Given
	userID := "user-123"
	
	// Create a prompt
	prompt := &entities.Prompt{
		ID:             "prompt-123",
		UserID:         userID,
		Name:           "existing-prompt",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Template",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := suite.promptRepo.Create(context.Background(), prompt)
	require.NoError(suite.T(), err)

	// When/Then - Test existing prompt
	isValid := suite.handler.ValidatePromptReference(context.Background(), userID, "existing-prompt")
	assert.True(suite.T(), isValid)

	// When/Then - Test non-existing prompt
	isValid = suite.handler.ValidatePromptReference(context.Background(), userID, "non-existing")
	assert.False(suite.T(), isValid)
}
