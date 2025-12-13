package usecases

import (
	"context"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
)

// MockUserRepository is a mock implementation of interfaces.UserRepository
type MockUserRepository struct {
	FindByIDFunc func(ctx context.Context, userID string) (*entities.User, error)
}

func (m *MockUserRepository) FindByID(ctx context.Context, userID string) (*entities.User, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockUserRepository) Create(ctx context.Context, user *entities.User) (string, error) {
	return "", nil
}

func (m *MockUserRepository) Update(ctx context.Context, userID string, updates map[string]interface{}) error {
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, userID string) error {
	return nil
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*entities.User, error) {
	return nil, nil
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*entities.User, error) {
	return nil, nil
}

// MockIdeasRepository is a mock implementation of interfaces.IdeasRepository
type MockIdeasRepository struct {
	CreateBatchFunc   func(ctx context.Context, ideas []*entities.Idea) error
	ListByUserIDFunc  func(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error)
	CountByUserIDFunc func(ctx context.Context, userID string) (int64, error)
	ClearByUserIDFunc func(ctx context.Context, userID string) error
}

func (m *MockIdeasRepository) CreateBatch(ctx context.Context, ideas []*entities.Idea) error {
	if m.CreateBatchFunc != nil {
		return m.CreateBatchFunc(ctx, ideas)
	}
	return nil
}

func (m *MockIdeasRepository) ListByUserID(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID, topicID, limit)
	}
	return nil, nil
}

func (m *MockIdeasRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	if m.CountByUserIDFunc != nil {
		return m.CountByUserIDFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockIdeasRepository) ClearByUserID(ctx context.Context, userID string) error {
	if m.ClearByUserIDFunc != nil {
		return m.ClearByUserIDFunc(ctx, userID)
	}
	return nil
}

// MockDraftRepository is a mock implementation of interfaces.DraftRepository
type MockDraftRepository struct {
	CreateFunc                 func(ctx context.Context, draft *entities.Draft) (string, error)
	FindByIDFunc               func(ctx context.Context, draftID string) (*entities.Draft, error)
	UpdateFunc                 func(ctx context.Context, draftID string, updates map[string]interface{}) error
	DeleteFunc                 func(ctx context.Context, draftID string) error
	ListByUserIDFunc           func(ctx context.Context, userID string, status entities.DraftStatus, draftType entities.DraftType) ([]*entities.Draft, error)
	UpdateStatusFunc           func(ctx context.Context, draftID string, status entities.DraftStatus) error
	AppendRefinementFunc       func(ctx context.Context, draftID string, entry entities.RefinementEntry) error
	FindReadyForPublishingFunc func(ctx context.Context, userID string) ([]*entities.Draft, error)
}

func (m *MockDraftRepository) Create(ctx context.Context, draft *entities.Draft) (string, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, draft)
	}
	return draft.ID, nil
}

func (m *MockDraftRepository) FindByID(ctx context.Context, draftID string) (*entities.Draft, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, draftID)
	}
	return nil, nil
}

func (m *MockDraftRepository) Update(ctx context.Context, draftID string, updates map[string]interface{}) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, draftID, updates)
	}
	return nil
}

func (m *MockDraftRepository) Delete(ctx context.Context, draftID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, draftID)
	}
	return nil
}

func (m *MockDraftRepository) ListByUserID(ctx context.Context, userID string, status entities.DraftStatus, draftType entities.DraftType) ([]*entities.Draft, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID, status, draftType)
	}
	return nil, nil
}

func (m *MockDraftRepository) UpdateStatus(ctx context.Context, draftID string, status entities.DraftStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, draftID, status)
	}
	return nil
}

func (m *MockDraftRepository) AppendRefinement(ctx context.Context, draftID string, entry entities.RefinementEntry) error {
	if m.AppendRefinementFunc != nil {
		return m.AppendRefinementFunc(ctx, draftID, entry)
	}
	return nil
}

func (m *MockDraftRepository) FindReadyForPublishing(ctx context.Context, userID string) ([]*entities.Draft, error) {
	if m.FindReadyForPublishingFunc != nil {
		return m.FindReadyForPublishingFunc(ctx, userID)
	}
	return nil, nil
}

// MockLLMService is a mock implementation of interfaces.LLMService
type MockLLMService struct {
	SendRequestFunc    func(ctx context.Context, prompt string) (string, error)
	GenerateIdeasFunc  func(ctx context.Context, topic string, count int) ([]string, error)
	GenerateDraftsFunc func(ctx context.Context, idea string, userContext string) (interfaces.DraftSet, error)
	RefineDraftFunc    func(ctx context.Context, draft string, userPrompt string, history []string) (string, error)
}

func (m *MockLLMService) SendRequest(ctx context.Context, prompt string) (string, error) {
	if m.SendRequestFunc != nil {
		return m.SendRequestFunc(ctx, prompt)
	}
	return "", nil
}

func (m *MockLLMService) GenerateIdeas(ctx context.Context, topic string, count int) ([]string, error) {
	if m.GenerateIdeasFunc != nil {
		return m.GenerateIdeasFunc(ctx, topic, count)
	}
	return nil, nil
}

func (m *MockLLMService) GenerateDrafts(ctx context.Context, idea string, userContext string) (interfaces.DraftSet, error) {
	if m.GenerateDraftsFunc != nil {
		return m.GenerateDraftsFunc(ctx, idea, userContext)
	}
	return interfaces.DraftSet{}, nil
}

func (m *MockLLMService) RefineDraft(ctx context.Context, draft string, userPrompt string, history []string) (string, error) {
	if m.RefineDraftFunc != nil {
		return m.RefineDraftFunc(ctx, draft, userPrompt, history)
	}
	return "", nil
}

// MockLinkedInService is a mock implementation of interfaces.LinkedInService
type MockLinkedInService struct {
	PublishPostFunc    func(ctx context.Context, content string, accessToken string) (*interfaces.LinkedInPostResponse, error)
	PublishArticleFunc func(ctx context.Context, title string, content string, accessToken string) (*interfaces.LinkedInPostResponse, error)
	ValidateTokenFunc  func(ctx context.Context, accessToken string) (bool, error)
	RefreshTokenFunc   func(ctx context.Context, refreshToken string) (string, error)
}

func (m *MockLinkedInService) PublishPost(ctx context.Context, content string, accessToken string) (*interfaces.LinkedInPostResponse, error) {
	if m.PublishPostFunc != nil {
		return m.PublishPostFunc(ctx, content, accessToken)
	}
	return nil, nil
}

func (m *MockLinkedInService) PublishArticle(ctx context.Context, title string, content string, accessToken string) (*interfaces.LinkedInPostResponse, error) {
	if m.PublishArticleFunc != nil {
		return m.PublishArticleFunc(ctx, title, content, accessToken)
	}
	return nil, nil
}

func (m *MockLinkedInService) ValidateToken(ctx context.Context, accessToken string) (bool, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(ctx, accessToken)
	}
	return false, nil
}

func (m *MockLinkedInService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	if m.RefreshTokenFunc != nil {
		return m.RefreshTokenFunc(ctx, refreshToken)
	}
	return "", nil
}
