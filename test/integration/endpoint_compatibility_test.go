package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories"
	"github.com/linkgen-ai/backend/src/interfaces/handlers"
	"github.com/linkgen-ai/backend/src/interfaces/routes"
)

// TestExistingEndpointsCompatibility tests that existing endpoints continue to work
// after the prompt system refactor
func TestExistingEndpointsCompatibility(tt *testing.T) {
	ctx := context.Background()

	tt.Run("GET /v1/prompts/{userId} should return prompts with new structure", func(t *testing.T) {
		// GIVEN existing prompts in database
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		promptRepo := repositories.NewPromptRepository(db)

		// Create prompts with different types
		prompts := []entities.Prompt{
			{
				ID:             primitive.NewObjectID().Hex(),
				UserID:         userID,
				Name:           "base1",
				Type:           entities.PromptTypeIdeas,
				PromptTemplate: "Generate {ideas} ideas about {name}",
				Active:         true,
				CreatedAt:      now(),
				UpdatedAt:      now(),
			},
			{
				ID:             primitive.NewObjectID().Hex(),
				UserID:         userID,
				Name:           "professional",
				Type:           entities.PromptTypeDrafts,
				PromptTemplate: "Create professional content about {topic_name}",
				Active:         true,
				CreatedAt:      now(),
				UpdatedAt:      now(),
			},
		}

		for _, prompt := range prompts {
			err := promptRepo.Create(ctx, &prompt)
			require.NoError(t, err)
		}

		// WHEN calling the existing endpoint
		router := setupTestRouter(db)
		req := httptest.NewRequest("GET", fmt.Sprintf("/v1/prompts/%s", userID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// THEN should return the expected response format
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response struct {
			Count   int                 `json:"count"`
			Prompts []entities.Prompt   `json:"prompts"`
		}
		
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, len(prompts), response.Count)
		assert.Len(t, response.Prompts, len(prompts))
		
		// This will fail until the endpoint is properly implemented
		t.Fatal("implement GET /v1/prompts/{userId} endpoint compatibility - FAILING IN TDD RED PHASE")
	})

	tt.Run("GET /v1/topics/{userId} should return topics with prompt references", func(t *testing.T) {
		// GIVEN topics with prompt references
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		topicRepo := repositories.NewTopicRepository(db)
		promptRepo := repositories.NewPromptRepository(db)

		// Create base prompt
		basePrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Generate {ideas} ideas about {name} with {[related_topics]}",
			Active:         true,
			CreatedAt:      now(),
			UpdatedAt:      now(),
		}

		err := promptRepo.Create(ctx, basePrompt)
		require.NoError(t, err)

		// Create topics with prompt references
		topics := []entities.Topic{
			{
				ID:             primitive.NewObjectID().Hex(),
				UserID:         userID,
				Name:           "TypeScript Patterns",
				Description:    "Advanced TypeScript techniques",
				Keywords:       []string{"typescript", "types", "generics"},
				Category:       "Frontend",
				Priority:       8,
				IdeasCount:     5,
				Active:         true,
				PromptName:     "base1",
				RelatedTopics:  []string{"JavaScript", "Node.js"},
				CreatedAt:      now(),
				UpdatedAt:      now(),
			},
			{
				ID:             primitive.NewObjectID().Hex(),
				UserID:         userID,
				Name:           "Go Microservices",
				Description:    "Building microservices with Go",
				Keywords:       []string{"go", "microservices", "grpc"},
				Category:       "Backend",
				Priority:       9,
				IdeasCount:     3,
				Active:         true,
				PromptName:     "base1",
				RelatedTopics:  []string{"Docker", "Kubernetes"},
				CreatedAt:      now(),
				UpdatedAt:      now(),
			},
		}

		for _, topic := range topics {
			_, err := topicRepo.Create(ctx, &topic)
			require.NoError(t, err)
		}

		// WHEN calling the existing endpoint
		router := setupTestRouter(db)
		req := httptest.NewRequest("GET", fmt.Sprintf("/v1/topics/%s", userID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// THEN should return topics with all expected fields
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response struct {
			Count  int               `json:"count"`
			Topics []entities.Topic  `json:"topics"`
		}
		
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, len(topics), response.Count)
		assert.Len(t, response.Topics, len(topics))
		
		// Verify prompt references are present
		for _, topic := range response.Topics {
			assert.NotEmpty(t, topic.PromptName, "Topic should have prompt reference")
			assert.Equal(t, "base1", topic.PromptName)
		}

		// This will fail until the endpoint is properly implemented
		t.Fatal("implement GET /v1/topics/{userId} endpoint with prompt references - FAILING IN TDD RED PHASE")
	})

	tt.Run("GET /v1/ideas/{userId} should return ideas with topic names", func(t *testing.T) {
		// GIVEN ideas with topic references
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		topicRepo := repositories.NewTopicRepository(db)
		ideaRepo := repositories.NewIdeaRepository(db)

		// Create topic
		topic := &entities.Topic{
			ID:          primitive.NewObjectID().Hex(),
			UserID:      userID,
			Name:        "React Testing",
			Description: "Testing React applications",
			CreatedAt:   now(),
			UpdatedAt:   now(),
		}

		createdTopic, err := topicRepo.Create(ctx, topic)
		require.NoError(t, err)

		// Create ideas for the topic
		ideas := []entities.Idea{
			{
				ID:        primitive.NewObjectID().Hex(),
				Content:   "Testing custom hooks with Jest and React Testing Library",
				TopicID:   createdTopic.ID,
				TopicName: createdTopic.Name, // New field from entity.md
				UserID:    userID,
				Used:      false,
				CreatedAt: now(),
				UpdatedAt: now(),
			},
			{
				ID:        primitive.NewObjectID().Hex(),
				Content:   "Integration testing for React components with Cypress",
				TopicID:   createdTopic.ID,
				TopicName: createdTopic.Name,
				UserID:    userID,
				Used:      true,
				CreatedAt: now(),
				UpdatedAt: now(),
			},
		}

		for _, idea := range ideas {
			err := ideaRepo.Create(ctx, &idea)
			require.NoError(t, err)
		}

		// WHEN calling the existing endpoint
		router := setupTestRouter(db)
		req := httptest.NewRequest("GET", fmt.Sprintf("/v1/ideas/%s", userID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// THEN should return ideas with topic names
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response struct {
			Count  int              `json:"count"`
			Ideas  []entities.Idea  `json:"ideas"`
		}
		
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, len(ideas), response.Count)
		assert.Len(t, response.Ideas, len(ideas))
		
		// Verify topic names are included
		for _, idea := range response.Ideas {
			assert.NotEmpty(t, idea.TopicName, "Idea should include topic name")
			assert.Equal(t, "React Testing", idea.TopicName)
		}

		// This will fail until the endpoint is properly implemented
		t.Fatal("implement GET /v1/ideas/{userId} endpoint with topic names - FAILING IN TDD RED PHASE")
	})

	tt.Run("POST /v1/drafts/generate should use prompt system", func(t *testing.T) {
		// GIVEN idea and prompt configuration
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		ideaID := primitive.NewObjectID().Hex()

		// Create idea
		idea := &entities.Idea{
			ID:        ideaID,
			Content:   "Optimizando el rendimiento de aplicaciones React con Memo",
			TopicName: "React Performance",
			UserID:    userID,
		}

		ideaRepo := repositories.NewIdeaRepository(db)
		err := ideaRepo.Create(ctx, idea)
		require.NoError(t, err)

		// WHEN calling the draft generation endpoint
		router := setupTestRouter(db)
		requestBody := map[string]interface{}{
			"user_id": userID,
			"idea_id": ideaID,
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/v1/drafts/generate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// THEN should accept request and return job ID
		assert.Equal(t, http.StatusAccepted, w.Code)
		
		var response struct {
			Message string `json:"message"`
			JobID   string `json:"job_id"`
		}
		
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "Draft generation started", response.Message)
		assert.NotEmpty(t, response.JobID, "Should return job ID for async processing")

		// This will fail until the endpoint is properly implemented
		t.Fatal("implement POST /v1/drafts/generate with prompt system - FAILING IN TDD RED PHASE")
	})

	tt.Run("GET /v1/drafts/jobs/{jobId} should return job status", func(t *testing.T) {
		// GIVEN a job ID from draft generation
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)
		
		jobID := primitive.NewObjectID().Hex()

		// WHEN checking job status
		router := setupTestRouter(db)
		req := httptest.NewRequest("GET", fmt.Sprintf("/v1/drafts/jobs/%s", jobID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// THEN should return job status
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response struct {
			JobID      string    `json:"job_id"`
			Status     string    `json:"status"`
			IdeaID     string    `json:"idea_id"`
			CreatedAt  string    `json:"created_at"`
			DraftIDs   []string  `json:"draft_ids,omitempty"`
			Error      string    `json:"error,omitempty"`
		}
		
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, jobID, response.JobID)
		assert.NotEmpty(t, response.Status, "Job should have a status")

		// This will fail until the endpoint is properly implemented
		t.Fatal("implement GET /v1/drafts/jobs/{jobId} endpoint - FAILING IN TDD RED PHASE")
	})
}

// TestResponseFormatCompatibility tests that response formats match existing contracts
func TestResponseFormatCompatibility(tt *testing.T) {
	tt.Run("should maintain backward compatibility in JSON responses", func(t *testing.T) {
		// GIVEN expected response structure
		expectedPromptResponse := `{
			"count": 2,
			"prompts": [
				{
					"id": "string",
					"name": "string",
					"type": "ideas|drafts",
					"prompt_template": "string",
					"active": true,
					"user_id": "string",
					"created_at": "timestamp",
					"updated_at": "timestamp"
				}
			]
		}`
		
		expectedTopicResponse := `{
			"count": 4,
			"topics": [
				{
					"id": "string",
					"user_id": "string",
					"name": "string",
					"description": "string",
					"keywords": ["string"],
					"category": "string",
					"priority": 5,
					"ideas": 3,
					"active": true,
					"prompt": "base1",
					"related_topics": ["string"],
					"created_at": "timestamp",
					"updated_at": "timestamp"
				}
			]
		}`
		
		// WHEN implementing endpoints
		// This will fail until endpoints return proper JSON structure
		t.Fatal("implement backward compatible JSON response formats - FAILING IN TDD RED PHASE")
		
		// THEN should match expected contracts
	})
}

// now returns current time for testing
func now() time.Time {
	return time.Now()
}

// setupTestRouter creates a test router with all routes
func setupTestRouter(db *mongo.Database) http.Handler {
	// This will fail until the router setup is implemented
	t.Fatal("implement test router setup - FAILING IN TDD RED PHASE")
	return nil
}
