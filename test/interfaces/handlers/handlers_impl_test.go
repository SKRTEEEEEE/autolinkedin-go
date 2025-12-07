package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/linkgen-ai/backend/src/application/usecases"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/valueobjects"
	"github.com/linkgen-ai/backend/src/interfaces/handlers"
	"go.uber.org/zap"
)

// Mock repositories and services for testing

type mockIdeasRepository struct {
	ideas map[string][]*entities.Idea
}

func newMockIdeasRepository() *mockIdeasRepository {
	return &mockIdeasRepository{
		ideas: make(map[string][]*entities.Idea),
	}
}

func (m *mockIdeasRepository) CreateBatch(ctx context.Context, ideas []*entities.Idea) error {
	if len(ideas) == 0 {
		return nil
	}
	userID := ideas[0].UserID
	if m.ideas[userID] == nil {
		m.ideas[userID] = make([]*entities.Idea, 0)
	}
	m.ideas[userID] = append(m.ideas[userID], ideas...)
	return nil
}

func (m *mockIdeasRepository) ListByUserID(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
	ideas, ok := m.ideas[userID]
	if !ok {
		return []*entities.Idea{}, nil
	}

	result := make([]*entities.Idea, 0)
	for _, idea := range ideas {
		if topicID != "" && idea.TopicID != topicID {
			continue
		}
		result = append(result, idea)
		if limit > 0 && len(result) >= limit {
			break
		}
	}

	return result, nil
}

func (m *mockIdeasRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	ideas, ok := m.ideas[userID]
	if !ok {
		return 0, nil
	}
	return int64(len(ideas)), nil
}

func (m *mockIdeasRepository) ClearByUserID(ctx context.Context, userID string) error {
	delete(m.ideas, userID)
	return nil
}

type mockUserRepository struct {
	users map[string]*entities.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*entities.User),
	}
}

func (m *mockUserRepository) FindByID(ctx context.Context, userID string) (*entities.User, error) {
	user, ok := m.users[userID]
	if !ok {
		return nil, nil
	}
	return user, nil
}

func (m *mockUserRepository) Create(ctx context.Context, user *entities.User) (string, error) {
	m.users[user.ID] = user
	return user.ID, nil
}

func (m *mockUserRepository) Update(ctx context.Context, userID string, updates map[string]interface{}) error {
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, userID string) error {
	delete(m.users, userID)
	return nil
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	return nil, nil
}

type mockDraftRepository struct {
	drafts map[string]*entities.Draft
}

func newMockDraftRepository() *mockDraftRepository {
	return &mockDraftRepository{
		drafts: make(map[string]*entities.Draft),
	}
}

func (m *mockDraftRepository) Create(ctx context.Context, draft *entities.Draft) (string, error) {
	m.drafts[draft.ID] = draft
	return draft.ID, nil
}

func (m *mockDraftRepository) FindByID(ctx context.Context, draftID string) (*entities.Draft, error) {
	draft, ok := m.drafts[draftID]
	if !ok {
		return nil, nil
	}
	return draft, nil
}

func (m *mockDraftRepository) Update(ctx context.Context, draftID string, updates map[string]interface{}) error {
	return nil
}

func (m *mockDraftRepository) Delete(ctx context.Context, draftID string) error {
	delete(m.drafts, draftID)
	return nil
}

func (m *mockDraftRepository) ListByUserID(ctx context.Context, userID string, status valueobjects.DraftStatus, draftType valueobjects.DraftType) ([]*entities.Draft, error) {
	result := make([]*entities.Draft, 0)
	for _, draft := range m.drafts {
		if draft.UserID != userID {
			continue
		}
		if status != "" && draft.Status != entities.DraftStatus(status) {
			continue
		}
		if draftType != "" && draft.Type != entities.DraftType(draftType) {
			continue
		}
		result = append(result, draft)
	}
	return result, nil
}

func (m *mockDraftRepository) UpdateStatus(ctx context.Context, draftID string, status entities.DraftStatus) error {
	return nil
}

func (m *mockDraftRepository) AppendRefinement(ctx context.Context, draftID string, entry entities.RefinementEntry) error {
	return nil
}

func (m *mockDraftRepository) FindReadyForPublishing(ctx context.Context, userID string) ([]*entities.Draft, error) {
	return []*entities.Draft{}, nil
}

type mockNATSPublisher struct {
	messages []interface{}
}

func newMockNATSPublisher() *mockNATSPublisher {
	return &mockNATSPublisher{
		messages: make([]interface{}, 0),
	}
}

func (m *mockNATSPublisher) Publish(ctx context.Context, data interface{}) error {
	m.messages = append(m.messages, data)
	return nil
}

type mockLLMService struct{}

func (m *mockLLMService) RefineDraft(ctx context.Context, content, prompt string, history []string) (string, error) {
	return content + " [refined]", nil
}

// Test cases

func TestHandlers_IdeasGetIdeas_Success(t *testing.T) {
	// Setup
	logger, _ := zap.NewDevelopment()
	ideasRepo := newMockIdeasRepository()
	userRepo := newMockUserRepository()

	// Add test user
	user := &entities.User{
		ID:    "675337baf901e2d790aabbcc",
		Email: "test@example.com",
	}
	userRepo.users[user.ID] = user

	// Add test ideas
	ideas := []*entities.Idea{
		{
			ID:      "idea1",
			UserID:  user.ID,
			TopicID: "topic1",
			Content: "Test idea 1",
		},
		{
			ID:      "idea2",
			UserID:  user.ID,
			TopicID: "topic1",
			Content: "Test idea 2",
		},
	}
	ideasRepo.CreateBatch(context.Background(), ideas)

	// Create use case and handler
	listUseCase := usecases.NewListIdeasUseCase(userRepo, ideasRepo)
	clearUseCase := usecases.NewClearIdeasUseCase(userRepo, ideasRepo)
	handler := handlers.NewIdeasHandler(listUseCase, clearUseCase, logger)

	// Create router
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Test request
	req := httptest.NewRequest(http.MethodGet, "/v1/ideas/675337baf901e2d790aabbcc", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["count"] == nil {
		t.Error("Expected 'count' field in response")
	}
}

func TestHandlers_IdeasClearIdeas_Success(t *testing.T) {
	// Setup
	logger, _ := zap.NewDevelopment()
	ideasRepo := newMockIdeasRepository()
	userRepo := newMockUserRepository()

	// Add test user
	user := &entities.User{
		ID:    "675337baf901e2d790aabbcc",
		Email: "test@example.com",
	}
	userRepo.users[user.ID] = user

	// Create use case and handler
	listUseCase := usecases.NewListIdeasUseCase(userRepo, ideasRepo)
	clearUseCase := usecases.NewClearIdeasUseCase(userRepo, ideasRepo)
	handler := handlers.NewIdeasHandler(listUseCase, clearUseCase, logger)

	// Create router
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Test request
	req := httptest.NewRequest(http.MethodDelete, "/v1/ideas/675337baf901e2d790aabbcc/clear", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

func TestHandlers_DraftsGenerateDrafts_Success(t *testing.T) {
	// Setup
	logger, _ := zap.NewDevelopment()
	draftRepo := newMockDraftRepository()
	publisher := newMockNATSPublisher()
	llmService := &mockLLMService{}

	// Create use case and handler
	refineUseCase := usecases.NewRefineDraftUseCase(draftRepo, llmService)
	handler := handlers.NewDraftsHandler(refineUseCase, draftRepo, publisher, logger)

	// Create router
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Test request
	requestBody := map[string]interface{}{
		"user_id": "675337baf901e2d790aabbcc",
		"idea_id": "675337baf901e2d790aabbdd",
	}
	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/drafts/generate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["job_id"] == nil {
		t.Error("Expected 'job_id' field in response")
	}

	if len(publisher.messages) == 0 {
		t.Error("Expected message to be published to NATS")
	}
}

func TestHandlers_ValidationObjectID(t *testing.T) {
	// Setup
	logger, _ := zap.NewDevelopment()
	ideasRepo := newMockIdeasRepository()
	userRepo := newMockUserRepository()

	listUseCase := usecases.NewListIdeasUseCase(userRepo, ideasRepo)
	clearUseCase := usecases.NewClearIdeasUseCase(userRepo, ideasRepo)
	handler := handlers.NewIdeasHandler(listUseCase, clearUseCase, logger)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Test request with invalid ObjectID
	req := httptest.NewRequest(http.MethodGet, "/v1/ideas/invalid-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ObjectID, got %d", w.Code)
	}
}
