package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/application/usecases"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
)

// TestGenerateDraftsUseCase_Success validates successful draft generation flow
func TestGenerateDraftsUseCase_Success(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	userRepo := &MockUserRepository{
		FindByIDFunc: func(ctx context.Context, userID string) (*entities.User, error) {
			user, _ := entities.NewUser(userID, "testuser", "es")
			user.Configuration = map[string]interface{}{
				"name":            "Test User",
				"expertise":       "Software Engineering",
				"tone_preference": "professional",
			}
			return user, nil
		},
	}

	ideasRepo := &MockIdeasRepository{
		ListByUserIDFunc: func(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
			idea := entities.NewIdea("675337baf901e2d790aabbdd", userID, "topic123", "Test idea content for Clean Architecture", nil)
			return []*entities.Idea{idea}, nil
		},
	}

	draftRepo := &MockDraftRepository{
		CreateFunc: func(ctx context.Context, draft *entities.Draft) (string, error) {
			return draft.ID, nil
		},
	}

	llmService := &MockLLMService{
		GenerateDraftsFunc: func(ctx context.Context, idea string, userContext string) (interfaces.DraftSet, error) {
			return interfaces.DraftSet{
				Posts: []string{
					"Post 1: Clean Architecture principles explained in detail",
					"Post 2: Dependency injection benefits for maintainability",
					"Post 3: Repository pattern explained with examples",
					"Post 4: Use cases in action for clean code",
					"Post 5: Testing strategies for better quality",
				},
				Articles: []string{
					"# Clean Architecture Guide\n\nThis is a comprehensive article about Clean Architecture patterns and how to apply them in real-world projects.",
				},
			}, nil
		},
	}

	// Create use case
	uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

	// Execute
	input := usecases.GenerateDraftsInput{
		UserID: "675337baf901e2d790aabbcc",
		IdeaID: "675337baf901e2d790aabbdd",
	}
	drafts, err := uc.Execute(ctx, input)

	// Validate
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(drafts) != 6 {
		t.Errorf("Expected 6 drafts (5 posts + 1 article), got %d", len(drafts))
	}

	// Count posts and articles
	postCount := 0
	articleCount := 0
	for _, draft := range drafts {
		if draft.Type == "POST" {
			postCount++
		} else if draft.Type == "ARTICLE" {
			articleCount++
		}
	}

	if postCount != 5 {
		t.Errorf("Expected 5 posts, got %d", postCount)
	}

	if articleCount != 1 {
		t.Errorf("Expected 1 article, got %d", articleCount)
	}
}

// TestGenerateDraftsUseCase_ValidationErrors validates input validation
func TestGenerateDraftsUseCase_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	// Create minimal valid mocks (won't be called)
	userRepo := &MockUserRepository{}
	ideasRepo := &MockIdeasRepository{}
	draftRepo := &MockDraftRepository{}
	llmService := &MockLLMService{}

	uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

	tests := []struct {
		name    string
		userID  string
		ideaID  string
		wantErr bool
	}{
		{
			name:    "error on empty user ID",
			userID:  "",
			ideaID:  "675337baf901e2d790aabbdd",
			wantErr: true,
		},
		{
			name:    "error on empty idea ID",
			userID:  "675337baf901e2d790aabbcc",
			ideaID:  "",
			wantErr: true,
		},
		{
			name:    "error on both empty",
			userID:  "",
			ideaID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := usecases.GenerateDraftsInput{
				UserID: tt.userID,
				IdeaID: tt.ideaID,
			}
			_, err := uc.Execute(ctx, input)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestGenerateDraftsUseCase_UserNotFound validates user existence check
func TestGenerateDraftsUseCase_UserNotFound(t *testing.T) {
	ctx := context.Background()

	userRepo := &MockUserRepository{
		FindByIDFunc: func(ctx context.Context, userID string) (*entities.User, error) {
			return nil, errors.New("user not found")
		},
	}

	ideasRepo := &MockIdeasRepository{}
	draftRepo := &MockDraftRepository{}
	llmService := &MockLLMService{}

	uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

	input := usecases.GenerateDraftsInput{
		UserID: "675337baf901e2d790aabbcc",
		IdeaID: "675337baf901e2d790aabbdd",
	}
	_, err := uc.Execute(ctx, input)

	if err == nil {
		t.Error("Expected error when user not found")
	}
}

// TestGenerateDraftsUseCase_IdeaNotFound validates idea existence check
func TestGenerateDraftsUseCase_IdeaNotFound(t *testing.T) {
	ctx := context.Background()

	userRepo := &MockUserRepository{
		FindByIDFunc: func(ctx context.Context, userID string) (*entities.User, error) {
			user, _ := entities.NewUser(userID, "testuser", "es")
			return user, nil
		},
	}

	ideasRepo := &MockIdeasRepository{
		ListByUserIDFunc: func(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
			// Return empty list - idea not found
			return []*entities.Idea{}, nil
		},
	}

	draftRepo := &MockDraftRepository{}
	llmService := &MockLLMService{}

	uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

	input := usecases.GenerateDraftsInput{
		UserID: "675337baf901e2d790aabbcc",
		IdeaID: "675337baf901e2d790aabbdd",
	}
	_, err := uc.Execute(ctx, input)

	if err == nil {
		t.Error("Expected error when idea not found")
	}
}

// TestGenerateDraftsUseCase_IdeaOwnership validates idea belongs to user
func TestGenerateDraftsUseCase_IdeaOwnership(t *testing.T) {
	ctx := context.Background()

	userRepo := &MockUserRepository{
		FindByIDFunc: func(ctx context.Context, userID string) (*entities.User, error) {
			user, _ := entities.NewUser(userID, "testuser", "es")
			return user, nil
		},
	}

	ideasRepo := &MockIdeasRepository{
		ListByUserIDFunc: func(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
			// Return idea belonging to different user
			idea := entities.NewIdea("675337baf901e2d790aabbdd", "different-user-id", "topic123", "Test idea", nil)
			return []*entities.Idea{idea}, nil
		},
	}

	draftRepo := &MockDraftRepository{}
	llmService := &MockLLMService{}

	uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

	input := usecases.GenerateDraftsInput{
		UserID: "675337baf901e2d790aabbcc",
		IdeaID: "675337baf901e2d790aabbdd",
	}
	_, err := uc.Execute(ctx, input)

	if err == nil {
		t.Error("Expected error when idea doesn't belong to user")
	}
}

// TestGenerateDraftsUseCase_IdeaAlreadyUsed validates idea can only be used once
func TestGenerateDraftsUseCase_IdeaAlreadyUsed(t *testing.T) {
	ctx := context.Background()

	userRepo := &MockUserRepository{
		FindByIDFunc: func(ctx context.Context, userID string) (*entities.User, error) {
			user, _ := entities.NewUser(userID, "testuser", "es")
			return user, nil
		},
	}

	ideasRepo := &MockIdeasRepository{
		ListByUserIDFunc: func(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
			idea := entities.NewIdea("675337baf901e2d790aabbdd", userID, "topic123", "Test idea", nil)
			// Mark idea as already used
			_ = idea.MarkAsUsed()
			return []*entities.Idea{idea}, nil
		},
	}

	draftRepo := &MockDraftRepository{}
	llmService := &MockLLMService{}

	uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

	input := usecases.GenerateDraftsInput{
		UserID: "675337baf901e2d790aabbcc",
		IdeaID: "675337baf901e2d790aabbdd",
	}
	_, err := uc.Execute(ctx, input)

	if err == nil {
		t.Error("Expected error when idea already used")
	}
}

// TestGenerateDraftsUseCase_IdeaExpired validates expired idea handling
func TestGenerateDraftsUseCase_IdeaExpired(t *testing.T) {
	ctx := context.Background()

	userRepo := &MockUserRepository{
		FindByIDFunc: func(ctx context.Context, userID string) (*entities.User, error) {
			user, _ := entities.NewUser(userID, "testuser", "es")
			return user, nil
		},
	}

	ideasRepo := &MockIdeasRepository{
		ListByUserIDFunc: func(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
			// Create expired idea
			expiresAt := time.Now().Add(-24 * time.Hour)
			idea := entities.NewIdea("675337baf901e2d790aabbdd", userID, "topic123", "Test idea", &expiresAt)
			return []*entities.Idea{idea}, nil
		},
	}

	draftRepo := &MockDraftRepository{}
	llmService := &MockLLMService{}

	uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

	input := usecases.GenerateDraftsInput{
		UserID: "675337baf901e2d790aabbcc",
		IdeaID: "675337baf901e2d790aabbdd",
	}
	_, err := uc.Execute(ctx, input)

	if err == nil {
		t.Error("Expected error when idea is expired")
	}
}

// TestGenerateDraftsUseCase_LLMErrors validates LLM error handling
func TestGenerateDraftsUseCase_LLMErrors(t *testing.T) {
	ctx := context.Background()

	userRepo := &MockUserRepository{
		FindByIDFunc: func(ctx context.Context, userID string) (*entities.User, error) {
			user, _ := entities.NewUser(userID, "testuser", "es")
			return user, nil
		},
	}

	ideasRepo := &MockIdeasRepository{
		ListByUserIDFunc: func(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
			idea := entities.NewIdea("675337baf901e2d790aabbdd", userID, "topic123", "Test idea", nil)
			return []*entities.Idea{idea}, nil
		},
	}

	draftRepo := &MockDraftRepository{}

	tests := []struct {
		name   string
		llmErr error
	}{
		{
			name:   "LLM service unavailable",
			llmErr: errors.New("connection refused"),
		},
		{
			name:   "LLM timeout",
			llmErr: errors.New("request timeout"),
		},
		{
			name:   "LLM invalid response",
			llmErr: errors.New("invalid JSON"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llmService := &MockLLMService{
				GenerateDraftsFunc: func(ctx context.Context, idea string, userContext string) (interfaces.DraftSet, error) {
					return interfaces.DraftSet{}, tt.llmErr
				},
			}

			uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

			input := usecases.GenerateDraftsInput{
				UserID: "675337baf901e2d790aabbcc",
				IdeaID: "675337baf901e2d790aabbdd",
			}
			_, err := uc.Execute(ctx, input)

			if err == nil {
				t.Error("Expected error when LLM fails")
			}
		})
	}
}

// TestGenerateDraftsUseCase_LLMInsufficientDrafts validates partial LLM responses
func TestGenerateDraftsUseCase_LLMInsufficientDrafts(t *testing.T) {
	ctx := context.Background()

	userRepo := &MockUserRepository{
		FindByIDFunc: func(ctx context.Context, userID string) (*entities.User, error) {
			user, _ := entities.NewUser(userID, "testuser", "es")
			return user, nil
		},
	}

	ideasRepo := &MockIdeasRepository{
		ListByUserIDFunc: func(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
			idea := entities.NewIdea("675337baf901e2d790aabbdd", userID, "topic123", "Test idea", nil)
			return []*entities.Idea{idea}, nil
		},
	}

	draftRepo := &MockDraftRepository{}

	tests := []struct {
		name          string
		postsCount    int
		articlesCount int
		wantErr       bool
	}{
		{
			name:          "error when LLM returns less than 5 posts",
			postsCount:    3,
			articlesCount: 1,
			wantErr:       true,
		},
		{
			name:          "error when LLM returns no articles",
			postsCount:    5,
			articlesCount: 0,
			wantErr:       true,
		},
		{
			name:          "error when LLM returns no drafts",
			postsCount:    0,
			articlesCount: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llmService := &MockLLMService{
				GenerateDraftsFunc: func(ctx context.Context, idea string, userContext string) (interfaces.DraftSet, error) {
					posts := make([]string, tt.postsCount)
					for i := 0; i < tt.postsCount; i++ {
						posts[i] = "Post content " + string(rune(i))
					}
					articles := make([]string, tt.articlesCount)
					for i := 0; i < tt.articlesCount; i++ {
						articles[i] = "Article content"
					}
					return interfaces.DraftSet{
						Posts:    posts,
						Articles: articles,
					}, nil
				},
			}

			uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

			input := usecases.GenerateDraftsInput{
				UserID: "675337baf901e2d790aabbcc",
				IdeaID: "675337baf901e2d790aabbdd",
			}
			_, err := uc.Execute(ctx, input)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestGenerateDraftsUseCase_RepositorySaveError validates error handling during save
func TestGenerateDraftsUseCase_RepositorySaveError(t *testing.T) {
	ctx := context.Background()

	userRepo := &MockUserRepository{
		FindByIDFunc: func(ctx context.Context, userID string) (*entities.User, error) {
			user, _ := entities.NewUser(userID, "testuser", "es")
			return user, nil
		},
	}

	ideasRepo := &MockIdeasRepository{
		ListByUserIDFunc: func(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
			idea := entities.NewIdea("675337baf901e2d790aabbdd", userID, "topic123", "Test idea", nil)
			return []*entities.Idea{idea}, nil
		},
	}

	draftRepo := &MockDraftRepository{
		CreateFunc: func(ctx context.Context, draft *entities.Draft) (string, error) {
			return "", errors.New("database connection lost")
		},
	}

	llmService := &MockLLMService{
		GenerateDraftsFunc: func(ctx context.Context, idea string, userContext string) (interfaces.DraftSet, error) {
			return interfaces.DraftSet{
				Posts: []string{
					"Post 1", "Post 2", "Post 3", "Post 4", "Post 5",
				},
				Articles: []string{
					"Article 1",
				},
			}, nil
		},
	}

	uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

	input := usecases.GenerateDraftsInput{
		UserID: "675337baf901e2d790aabbcc",
		IdeaID: "675337baf901e2d790aabbdd",
	}
	_, err := uc.Execute(ctx, input)

	if err == nil {
		t.Error("Expected error when repository fails")
	}
}

// TestGenerateDraftsUseCase_ContextCancellation validates context handling
func TestGenerateDraftsUseCase_ContextCancellation(t *testing.T) {
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	userRepo := &MockUserRepository{
		FindByIDFunc: func(ctx context.Context, userID string) (*entities.User, error) {
			// Check if context is cancelled
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			user, _ := entities.NewUser(userID, "testuser", "es")
			return user, nil
		},
	}

	ideasRepo := &MockIdeasRepository{}
	draftRepo := &MockDraftRepository{}
	llmService := &MockLLMService{}

	uc := usecases.NewGenerateDraftsUseCase(userRepo, ideasRepo, draftRepo, llmService)

	input := usecases.GenerateDraftsInput{
		UserID: "675337baf901e2d790aabbcc",
		IdeaID: "675337baf901e2d790aabbdd",
	}
	_, err := uc.Execute(ctx, input)

	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}
