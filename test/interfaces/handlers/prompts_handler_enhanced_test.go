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
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// NOTE: These tests are written according to TDD Red pattern - they will FAIL
// until the enhanced PromptHandler API implementation exists

func TestPromptsHandlerEnhanced(t *testing.T) {
	// Test for enhanced PromptHandler API endpoints:
	// - GetPromptByName: Get specific prompt by name
	// - CreateCustomPrompt: Create new custom prompt with name
	// - ResetUserPrompts: Reset all user prompts to defaults
	// - ValidatePromptTemplate: Validate template syntax endpoint
	// - ListDefaultPrompts: List available default prompts

	// Setup
	logger, _ := zap.NewDevelopment()
	mockPromptsRepo := &MockPromptsRepository{}
	mockUserRepo := &MockUserRepository{}

	handler := NewPromptsHandler(mockPromptsRepo, mockUserRepo, logger)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Register enhanced routes
	handler.RegisterEnhancedRoutes(router)

	ctx := context.Background()
	userID := "test-user-123"
	now := time.Now()

	setup := func() {
		mockPromptsRepo.prompts = []*entities.Prompt{}
		mockUserRepo.users = []*entities.User{
			{
				ID:        userID,
				Email:     "test@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
	}

	t.Run("should get prompt by name", func(t *testing.T) {
		setup()

		// GIVEN a user with a specific prompt
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "creative",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Genera {ideas} ideas creativas sobre {name}",
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		mockPromptsRepo.prompts = append(mockPromptsRepo.prompts, prompt)

		// WHEN making a request to get the prompt by name
		req := httptest.NewRequest(http.MethodGet, "/v1/prompts/"+userID+"/by-name/creative", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// THEN the response should contain the prompt
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		promptData := response["prompt"].(map[string]interface{})
		assert.Equal(t, prompt.ID, promptData["id"])
		assert.Equal(t, "creative", promptData["name"])
		assert.Equal(t, string(entities.PromptTypeIdeas), promptData["type"])
		assert.Contains(t, promptData["prompt_template"], "ideas creativas")
		assert.Equal(t, true, promptData["active"])
	})

	t.Run("should return 404 when prompt by name not found", func(t *testing.T) {
		setup()

		// GIVEN a user without the requested prompt
		// WHEN making a request to get a non-existent prompt
		req := httptest.NewRequest(http.MethodGet, "/v1/prompts/"+userID+"/by-name/nonexistent", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// THEN should return not found
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "NOT_FOUND", response["code"])
		assert.Contains(t, response["message"], "not found")
	})

	t.Run("should create custom prompt with name", func(t *testing.T) {
		setup()

		// GIVEN a request to create a new custom prompt
		createData := map[string]interface{}{
			"user_id":         userID,
			"name":            "my-custom-prompt",
			"type":            "ideas",
			"prompt_template": "Generate {ideas} innovative ideas about {name} with focus on {[related_topics]}",
		}

		jsonData, err := json.Marshal(createData)
		require.NoError(t, err)

		// WHEN making the request
		req := httptest.NewRequest(http.MethodPost, "/v1/prompts/custom", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// THEN the prompt should be created
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "my-custom-prompt", response["name"])
		assert.Equal(t, userID, response["user_id"])
		assert.Equal(t, "ideas", response["type"])
		assert.Contains(t, response["prompt_template"], "innovative ideas")
		assert.Equal(t, true, response["active"])

		// AND it should be stored in the repository
		assert.Len(t, mockPromptsRepo.prompts, 1)
		assert.Equal(t, "my-custom-prompt", mockPromptsRepo.prompts[0].Name)
	})

	t.Run("should validate prompt template variables", func(t *testing.T) {
		setup()

		// WHEN validating a template with correct variables
		validData := map[string]interface{}{
			"type":     "ideas",
			"template": "Generate {ideas} ideas about {name} using {[related_topics]}",
		}

		jsonData, _ := json.Marshal(validData)
		req := httptest.NewRequest(http.MethodPost, "/v1/prompts/validate", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// THEN should return valid
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["valid"])
		assert.Equal(t, []interface{}{"ideas", "name", "[related_topics]"}, response["variables"])
	})

	t.Run("should reject invalid prompt template", func(t *testing.T) {
		setup()

		// WHEN validating a template with missing required variables
		invalidData := map[string]interface{}{
			"type":     "ideas",
			"template": "Generate ideas about name without placeholders",
		}

		jsonData, _ := json.Marshal(invalidData)
		req := httptest.NewRequest(http.MethodPost, "/v1/prompts/validate", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// THEN should return invalid with error details
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, false, response["valid"])
		assert.Contains(t, response["message"], "missing required variables")
	})

	t.Run("should list available default prompts", func(t *testing.T) {
		setup()

		// GIVEN default prompts available in seed directory
		// WHEN requesting default prompts
		req := httptest.NewRequest(http.MethodGet, "/v1/prompts/defaults", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// THEN should return list of default prompt templates
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		defaults := response["defaults"].([]interface{})
		assert.Greater(t, len(defaults), 0)

		// Verify at least base1 and pro are listed
		var base1Found, proFound bool
		for _, item := range defaults {
			def := item.(map[string]interface{})
			if def["name"] == "base1" {
				base1Found = true
				assert.Equal(t, "ideas", def["type"])
				assert.Contains(t, def["template"], "Genera {ideas} ideas")
			}
			if def["name"] == "pro" {
				proFound = true
				assert.Equal(t, "drafts", def["type"])
				assert.Contains(t, def["template"], "Escribe un post profesional")
			}
		}
		assert.True(t, base1Found, "base1 default prompt should be found")
		assert.True(t, proFound, "pro default prompt should be found")
	})

	t.Run("should reset user prompts to defaults", func(t *testing.T) {
		setup()

		// GIVEN a user with custom prompts
		customPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Custom template",
			Active:         true,
			CreatedAt:      now.Add(-time.Hour),
			UpdatedAt:      now.Add(-time.Hour),
		}
		mockPromptsRepo.prompts = append(mockPromptsRepo.prompts, customPrompt)

		// WHEN resetting to defaults
		req := httptest.NewRequest(http.MethodPost, "/v1/prompts/"+userID+"/reset", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// THEN user prompts should be reset
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["reset"])
		assert.Greater(t, response["count"].(float64), 0)

		// AND verify base1 was reset to default content
		base1Prompt := mockPromptsRepo.FindByName(ctx, userID, "base1")
		require.NotNil(t, base1Prompt)
		assert.Contains(t, base1Prompt.PromptTemplate, "Genera {ideas} ideas")
		assert.False(t, base1Prompt.Active) // Should be inactive until explicitly activated
	})

	t.Run("should activate a specific prompt by name", func(t *testing.T) {
		setup()

		// GIVEN a user with an inactive prompt
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "creative",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Template for creative ideas",
			Active:         false,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		mockPromptsRepo.prompts = append(mockPromptsRepo.prompts, prompt)

		// WHEN activating the prompt
		req := httptest.NewRequest(http.MethodPost, "/v1/prompts/"+userID+"/activate/creative", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// THEN the prompt should be activated
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["activated"])
		assert.Equal(t, "creative", response["name"])

		// AND verify it's now active in the repository
		activePrompt := mockPromptsRepo.FindActiveByName(ctx, userID, "creative")
		require.NotNil(t, activePrompt)
		assert.True(t, activePrompt.Active)
	})

	t.Run("should deactivate a specific prompt by name", func(t *testing.T) {
		setup()

		// GIVEN a user with an active prompt
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "professional",
			Type:           entities.PromptTypeDrafts,
			PromptTemplate: "Professional template",
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		mockPromptsRepo.prompts = append(mockPromptsRepo.prompts, prompt)

		// WHEN deactivating the prompt
		req := httptest.NewRequest(http.MethodPost, "/v1/prompts/"+userID+"/deactivate/professional", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// THEN the prompt should be deactivated
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["deactivated"])
		assert.Equal(t, "professional", response["name"])

		// AND verify it's no longer active in the repository
		activePrompt := mockPromptsRepo.FindActiveByName(ctx, userID, "professional")
		assert.Nil(t, activePrompt)

		// Should still exist but be inactive
		inactivePrompt := mockPromptsRepo.FindByName(ctx, userID, "professional")
		require.NotNil(t, inactivePrompt)
		assert.False(t, inactivePrompt.Active)
	})

	t.Run("should validate duplicate prompt names for user", func(t *testing.T) {
		setup()

		// GIVEN a user with an existing prompt
		existing := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "duplicate-test",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Existing template",
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		mockPromptsRepo.prompts = append(mockPromptsRepo.prompts, existing)

		// WHEN trying to create another prompt with the same name
		createData := map[string]interface{}{
			"user_id":         userID,
			"name":            "duplicate-test",
			"type":            "ideas",
			"prompt_template": "New template with same name",
		}

		jsonData, _ := json.Marshal(createData)
		req := httptest.NewRequest(http.MethodPost, "/v1/prompts/custom", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// THEN should return conflict error
		assert.Equal(t, http.StatusConflict, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "CONFLICT", response["code"])
		assert.Contains(t, response["message"], "already exists")
	})
}

// NEW methods to be added to PromptsHandler

// RegisterEnhancedRoutes registers the new prompt management routes
func (h *PromptsHandler) RegisterEnhancedRoutes(router *mux.Router) {
	// TODO: Implementation needed - this test will fail until implemented
	// These routes need to be implemented:
	// - /v1/prompts/{userId}/by-name/{name}
	// - /v1/prompts/custom
	// - /v1/prompts/{userId}/reset
	// - /v1/prompts/validate
	// - /v1/prompts/defaults
	// - /v1/prompts/{userId}/activate/{name}
	// - /v1/prompts/{userId}/deactivate/{name}
}

// NEW methods for MockPromptsRepository to support enhanced tests

// FindByName finds a prompt by name and user
func (m *MockPromptsRepository) FindByName(ctx context.Context, userID string, name string) *entities.Prompt {
	for _, p := range m.prompts {
		if p.Name == name && p.UserID == userID {
			return p
		}
	}
	return nil
}

// FindActiveByName finds an active prompt by name and user
func (m *MockPromptsRepository) FindActiveByName(ctx context.Context, userID string, name string) *entities.Prompt {
	p := m.FindByName(ctx, userID, name)
	if p != nil && p.Active {
		return p
	}
	return nil
}

// FindByNameAndType finds a prompt by name, user, and type
func (m *MockPromptsRepository) FindByNameAndType(ctx context.Context, userID string, name string, promptType entities.PromptType) *entities.Prompt {
	for _, p := range m.prompts {
		if p.Name == name && p.UserID == userID && p.Type == promptType {
			return p
		}
	}
	return nil
}

// FindActiveByNameAndType finds an active prompt by name, user, and type
func (m *MockPromptsRepository) FindActiveByNameAndType(ctx context.Context, userID string, name string, promptType entities.PromptType) *entities.Prompt {
	p := m.FindByNameAndType(ctx, userID, name, promptType)
	if p != nil && p.Active {
		return p
	}
	return nil
}

// CreateWithName creates a prompt with a specific name
func (m *MockPromptsRepository) CreateWithName(ctx context.Context, prompt *entities.Prompt) error {
	prompt.ID = "mock-id-" + string(rune(len(m.prompts)))
	m.prompts = append(m.prompts, prompt)
	return nil
}

// UpdateActiveStatus updates the active status of a prompt
func (m *MockPromptsRepository) UpdateActiveStatus(ctx context.Context, userID string, name string, active bool) error {
	for i, p := range m.prompts {
		if p.Name == name && p.UserID == userID {
			m.prompts[i].Active = active
			m.prompts[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return assert.AnError
}
