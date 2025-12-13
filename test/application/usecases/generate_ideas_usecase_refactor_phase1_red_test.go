package usecases

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
)

// TDD Red tests for Issue 3: Refactorización Fase 1 (Generación de Ideas)
// These tests are expected to fail until the refactor tasks in docs/task/3-2-refactorizar-fase-1.md are implemented.

func TestGenerateIdeasForTopic_UsesTopicPromptAndIdeasCount(t *testing.T) {
	ctx := context.Background()

	user := &entities.User{
		ID:        "user-1",
		Email:     "user-1@example.com",
		Language:  "es",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	topic := &entities.Topic{
		ID:            "topic-1",
		UserID:        user.ID,
		Name:          "Growth Marketing",
		Ideas:         3,
		Prompt:        "base1",
		RelatedTopics: []string{"SEO", "Paid Media"},
		Active:        true,
		CreatedAt:     time.Now(),
	}

	promptTemplate := "Genera {ideas} ideas sobre {name} en {language}. Temas relacionados: {related_topics}"
	prompt := &entities.Prompt{
		ID:             "prompt-1",
		UserID:         user.ID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: promptTemplate,
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	llmResponse := `{"ideas": ["Idea 1", "Idea 2", "Idea 3"]}`

	userRepo := &stubUserRepo{user: user}
	topicRepo := &stubTopicRepo{topics: map[string]*entities.Topic{topic.ID: topic}}
	promptRepo := &stubPromptsRepo{promptsByName: map[string]*entities.Prompt{"base1": prompt}}
	ideaRepo := &stubIdeasRepo{}
	llm := &stubLLM{response: llmResponse}

	useCase := NewGenerateIdeasUseCase(userRepo, topicRepo, ideaRepo, promptRepo, llm)

	ideas, err := useCase.GenerateIdeasForTopic(ctx, topic.ID)

	if err != nil {
		t.Fatalf("expected ideas to be generated using topic prompt, got error: %v", err)
	}

	if len(ideas) != topic.Ideas {
		t.Fatalf("expected %d ideas, got %d", topic.Ideas, len(ideas))
	}

	if len(llm.prompts) == 0 || !strings.Contains(llm.prompts[0], "Genera 3 ideas") {
		t.Fatalf("expected LLM to receive processed prompt with ideas count and topic name")
	}

	if !ideaRepo.savedWithTopicName(topic.Name) {
		t.Fatalf("expected ideas to be persisted with topic name %s", topic.Name)
	}
}

func TestGenerateIdeasForTopic_FallbacksToActivePromptWhenNamedMissing(t *testing.T) {
	ctx := context.Background()

	user := &entities.User{
		ID:        "user-2",
		Email:     "user-2@example.com",
		Language:  "es",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	topic := &entities.Topic{
		ID:        "topic-2",
		UserID:    user.ID,
		Name:      "Product Analytics",
		Ideas:     2,
		Prompt:    "non-existent",
		Active:    true,
		CreatedAt: time.Now(),
	}

	fallbackPrompt := &entities.Prompt{
		ID:             "prompt-fallback",
		UserID:         user.ID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Genera {ideas} ideas concisas sobre {name} en {language}",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	llmResponse := `{"ideas": ["Idea A", "Idea B"]}`

	userRepo := &stubUserRepo{user: user}
	topicRepo := &stubTopicRepo{topics: map[string]*entities.Topic{topic.ID: topic}}
	promptRepo := &stubPromptsRepo{
		promptsByName: map[string]*entities.Prompt{},
		activeByType: map[entities.PromptType][]*entities.Prompt{
			entities.PromptTypeIdeas: {fallbackPrompt},
		},
	}
	ideaRepo := &stubIdeasRepo{}
	llm := &stubLLM{response: llmResponse}

	useCase := NewGenerateIdeasUseCase(userRepo, topicRepo, ideaRepo, promptRepo, llm)

	ideas, err := useCase.GenerateIdeasForTopic(ctx, topic.ID)

	if err != nil {
		t.Fatalf("expected fallback prompt to be used when topic prompt is missing, got error: %v", err)
	}

	if !promptRepo.fallbackUsed {
		t.Fatalf("expected prompts repository fallback to active prompt type to be used")
	}

	if len(ideas) != topic.Ideas {
		t.Fatalf("expected %d ideas using fallback prompt, got %d", topic.Ideas, len(ideas))
	}
}

func TestGenerateIdeasForTopic_TrimsIdeasOver200Chars(t *testing.T) {
	ctx := context.Background()

	user := &entities.User{
		ID:        "user-3",
		Email:     "user-3@example.com",
		Language:  "es",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	topic := &entities.Topic{
		ID:        "topic-3",
		UserID:    user.ID,
		Name:      "AI Safety",
		Ideas:     1,
		Prompt:    "base1",
		Active:    true,
		CreatedAt: time.Now(),
	}

	prompt := &entities.Prompt{
		ID:             "prompt-3",
		UserID:         user.ID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Genera {ideas} ideas sobre {name} con longitud máxima de 200 caracteres",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	longContent := strings.Repeat("a", entities.MaxIdeaContentLength+40)
	llmResponse := fmt.Sprintf(`{"ideas": ["%s"]}`, longContent)

	userRepo := &stubUserRepo{user: user}
	topicRepo := &stubTopicRepo{topics: map[string]*entities.Topic{topic.ID: topic}}
	promptRepo := &stubPromptsRepo{promptsByName: map[string]*entities.Prompt{"base1": prompt}}
	ideaRepo := &stubIdeasRepo{}
	llm := &stubLLM{response: llmResponse}

	useCase := NewGenerateIdeasUseCase(userRepo, topicRepo, ideaRepo, promptRepo, llm)

	ideas, err := useCase.GenerateIdeasForTopic(ctx, topic.ID)

	if err != nil {
		t.Fatalf("expected long ideas to be trimmed and saved instead of rejected, got error: %v", err)
	}

	if len(ideas) != 1 {
		t.Fatalf("expected one idea after trimming, got %d", len(ideas))
	}

	if len(ideas[0].Content) > entities.MaxIdeaContentLength {
		t.Fatalf("expected idea content to be trimmed to %d characters, got %d", entities.MaxIdeaContentLength, len(ideas[0].Content))
	}
}

// --- Test Doubles ---

type stubUserRepo struct {
	user *entities.User
}

func (s *stubUserRepo) Create(ctx context.Context, user *entities.User) (string, error) {
	return "", fmt.Errorf("Create not implemented")
}

func (s *stubUserRepo) FindByID(ctx context.Context, userID string) (*entities.User, error) {
	if s.user != nil && s.user.ID == userID {
		return s.user, nil
	}
	return nil, nil
}

func (s *stubUserRepo) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	return nil, fmt.Errorf("FindByEmail not implemented")
}

func (s *stubUserRepo) Update(ctx context.Context, userID string, updates map[string]interface{}) error {
	return fmt.Errorf("Update not implemented")
}

func (s *stubUserRepo) UpdateLinkedInToken(ctx context.Context, userID string, token string) error {
	return fmt.Errorf("UpdateLinkedInToken not implemented")
}

func (s *stubUserRepo) Delete(ctx context.Context, userID string) error {
	return fmt.Errorf("Delete not implemented")
}

type stubTopicRepo struct {
	topics map[string]*entities.Topic
}

func (s *stubTopicRepo) Create(ctx context.Context, topic *entities.Topic) (string, error) {
	if s.topics == nil {
		s.topics = make(map[string]*entities.Topic)
	}
	s.topics[topic.ID] = topic
	return topic.ID, nil
}

func (s *stubTopicRepo) FindByID(ctx context.Context, topicID string) (*entities.Topic, error) {
	if s.topics == nil {
		return nil, nil
	}
	return s.topics[topicID], nil
}

func (s *stubTopicRepo) ListByUserID(ctx context.Context, userID string) ([]*entities.Topic, error) {
	return nil, fmt.Errorf("ListByUserID not implemented")
}

func (s *stubTopicRepo) FindRandomByUserID(ctx context.Context, userID string) (*entities.Topic, error) {
	for _, topic := range s.topics {
		if topic.UserID == userID {
			return topic, nil
		}
	}
	return nil, nil
}

func (s *stubTopicRepo) Update(ctx context.Context, topic *entities.Topic) error {
	return fmt.Errorf("Update not implemented")
}

func (s *stubTopicRepo) Delete(ctx context.Context, topicID string) error {
	return fmt.Errorf("Delete not implemented")
}

func (s *stubTopicRepo) FindByPrompt(ctx context.Context, userID string, promptName string) ([]*entities.Topic, error) {
	return nil, fmt.Errorf("FindByPrompt not implemented")
}

func (s *stubTopicRepo) FindByIdeasRange(ctx context.Context, userID string, minIdeas, maxIdeas int) ([]*entities.Topic, error) {
	return nil, fmt.Errorf("FindByIdeasRange not implemented")
}

type stubIdeasRepo struct {
	saved []*entities.Idea
}

func (s *stubIdeasRepo) CreateBatch(ctx context.Context, ideas []*entities.Idea) error {
	s.saved = append(s.saved, ideas...)
	return nil
}

func (s *stubIdeasRepo) ListByUserID(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
	return nil, fmt.Errorf("ListByUserID not implemented")
}

func (s *stubIdeasRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return 0, fmt.Errorf("CountByUserID not implemented")
}

func (s *stubIdeasRepo) ClearByUserID(ctx context.Context, userID string) error {
	return fmt.Errorf("ClearByUserID not implemented")
}

func (s *stubIdeasRepo) savedWithTopicName(name string) bool {
	for _, idea := range s.saved {
		if idea.TopicName != name {
			return false
		}
	}
	return len(s.saved) > 0
}

type stubPromptsRepo struct {
	promptsByName map[string]*entities.Prompt
	activeByType  map[entities.PromptType][]*entities.Prompt
	fallbackUsed  bool
}

func (s *stubPromptsRepo) Create(ctx context.Context, prompt *entities.Prompt) (string, error) {
	return "", fmt.Errorf("Create not implemented")
}

func (s *stubPromptsRepo) FindByID(ctx context.Context, id string) (*entities.Prompt, error) {
	return nil, fmt.Errorf("FindByID not implemented")
}

func (s *stubPromptsRepo) FindByName(ctx context.Context, userID string, name string) (*entities.Prompt, error) {
	if s.promptsByName == nil {
		return nil, nil
	}
	return s.promptsByName[name], nil
}

func (s *stubPromptsRepo) ListByUserID(ctx context.Context, userID string) ([]*entities.Prompt, error) {
	return nil, fmt.Errorf("ListByUserID not implemented")
}

func (s *stubPromptsRepo) ListByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	return nil, fmt.Errorf("ListByUserIDAndType not implemented")
}

func (s *stubPromptsRepo) FindActiveByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	s.fallbackUsed = true
	if s.activeByType == nil {
		return nil, nil
	}
	return s.activeByType[promptType], nil
}

func (s *stubPromptsRepo) FindByUserIDAndStyle(ctx context.Context, userID string, styleName string) (*entities.Prompt, error) {
	return nil, fmt.Errorf("FindByUserIDAndStyle not implemented")
}

func (s *stubPromptsRepo) Update(ctx context.Context, prompt *entities.Prompt) error {
	return fmt.Errorf("Update not implemented")
}

func (s *stubPromptsRepo) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("Delete not implemented")
}

func (s *stubPromptsRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return 0, fmt.Errorf("CountByUserID not implemented")
}

type stubLLM struct {
	prompts  []string
	response string
	err      error
}

func (s *stubLLM) SendRequest(ctx context.Context, prompt string) (string, error) {
	s.prompts = append(s.prompts, prompt)
	if s.err != nil {
		return "", s.err
	}
	return s.response, nil
}

func (s *stubLLM) GenerateIdeas(ctx context.Context, topic string, count int) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []string{"idea"}, nil
}

func (s *stubLLM) GenerateDrafts(ctx context.Context, idea string, userContext string) (interfaces.DraftSet, error) {
	return interfaces.DraftSet{}, fmt.Errorf("GenerateDrafts not implemented")
}

func (s *stubLLM) RefineDraft(ctx context.Context, draft string, userPrompt string, history []string) (string, error) {
	return "", fmt.Errorf("RefineDraft not implemented")
}
