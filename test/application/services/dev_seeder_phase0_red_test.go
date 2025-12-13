package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/application/services"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TDD Red tests for Issue 3 Phase 0 seeding behaviour

func TestDevSeeder_SeedDefaultPrompts_SyncsMissingFiles(t *testing.T) {
	ctx := context.Background()

	promptDir := t.TempDir()
	writePromptFile(t, promptDir, "base1.idea.md", "base1", "ideas", "Contenido base1")
	writePromptFile(t, promptDir, "custom.idea.md", "custom", "ideas", "Contenido custom")
	writePromptFile(t, promptDir, "legacy.old.md", "legacy", "ideas", "Debe ignorarse")

	promptRepo := newInMemoryPromptRepo()
	// Pretend one prompt already exists so CountByUserID > 0
	_ = promptRepo.Create(ctx, &entities.Prompt{
		ID:             "existing-1",
		UserID:         services.DevUserID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "ya existe",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	seeder := services.NewDevSeeder(
		&stubUserRepo{},
		&stubTopicRepo{},
		&stubIdeasRepo{},
		promptRepo,
		&stubLLM{},
		zap.NewNop(),
		&services.DevSeederConfig{PromptDir: promptDir},
	)

	err := seeder.SeedDefaultPrompts(ctx)
	require.NoError(t, err)

	// Should have created missing prompt from files even when prompts already existed
	prompts, err := promptRepo.ListByUserID(ctx, services.DevUserID)
	require.NoError(t, err)
	assert.Len(t, prompts, 2)
	assert.NotNil(t, promptRepo.mustFindByName(ctx, services.DevUserID, "custom"))
}

func TestDevSeeder_SeedDefaultTopics_UsesTopicJSONAndSetsPrompt(t *testing.T) {
	ctx := context.Background()

	promptDir := t.TempDir()
	topicSeed := filepath.Join(promptDir, "topic.json")
	topics := []map[string]interface{}{
		{
			"user_id":     services.DevUserID,
			"name":        "Cloud",
			"description": "Infra on cloud",
			"category":    "Tech",
			"priority":    6,
			"ideas":       3,
		},
		{
			"user_id":     services.DevUserID,
			"name":        "Data",
			"description": "Data eng",
			"category":    "Tech",
			"priority":    4,
			"ideas":       2,
		},
	}
	writeJSON(t, topicSeed, topics)

	topicRepo := newInMemoryTopicRepo()

	seeder := services.NewDevSeeder(
		&stubUserRepo{},
		topicRepo,
		&stubIdeasRepo{},
		newInMemoryPromptRepo(),
		&stubLLM{},
		zap.NewNop(),
		&services.DevSeederConfig{TopicSeedPath: topicSeed},
	)

	err := seeder.SeedDefaultTopics(ctx)
	require.NoError(t, err)

	savedTopics, err := topicRepo.ListByUserID(ctx, services.DevUserID)
	require.NoError(t, err)
	assert.Len(t, savedTopics, 2)
	for _, topic := range savedTopics {
		assert.Equal(t, entities.DefaultPrompt, topic.Prompt)
	}
}

// --- Helpers and stubs ---

func writePromptFile(t *testing.T, dir, filename, name, promptType, content string) {
	t.Helper()
	header := fmt.Sprintf("name: %s\ntype: %s\n", name, promptType)
	data := "---\n" + header + "---\n" + content
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(data), 0o644))
}

func writeJSON(t *testing.T, path string, data interface{}) {
	t.Helper()
	content, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, content, 0o644))
}

type inMemoryPromptRepo struct {
	prompts map[string]*entities.Prompt
}

func newInMemoryPromptRepo() *inMemoryPromptRepo {
	return &inMemoryPromptRepo{prompts: make(map[string]*entities.Prompt)}
}

func (r *inMemoryPromptRepo) Create(ctx context.Context, prompt *entities.Prompt) (string, error) {
	r.prompts[prompt.Name] = prompt
	return prompt.ID, nil
}

func (r *inMemoryPromptRepo) FindByID(ctx context.Context, id string) (*entities.Prompt, error) {
	return nil, nil
}

func (r *inMemoryPromptRepo) FindByName(ctx context.Context, userID string, name string) (*entities.Prompt, error) {
	return r.prompts[name], nil
}

func (r *inMemoryPromptRepo) ListByUserID(ctx context.Context, userID string) ([]*entities.Prompt, error) {
	var out []*entities.Prompt
	for _, p := range r.prompts {
		out = append(out, p)
	}
	return out, nil
}

func (r *inMemoryPromptRepo) ListByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	var out []*entities.Prompt
	for _, p := range r.prompts {
		if p.Type == promptType {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *inMemoryPromptRepo) FindActiveByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	return r.ListByUserIDAndType(ctx, userID, promptType)
}

func (r *inMemoryPromptRepo) FindByUserIDAndStyle(ctx context.Context, userID string, styleName string) (*entities.Prompt, error) {
	return nil, nil
}

func (r *inMemoryPromptRepo) Update(ctx context.Context, prompt *entities.Prompt) error { return nil }

func (r *inMemoryPromptRepo) Delete(ctx context.Context, id string) error { return nil }

func (r *inMemoryPromptRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return int64(len(r.prompts)), nil
}

func (r *inMemoryPromptRepo) mustFindByName(ctx context.Context, userID, name string) *entities.Prompt {
	p, _ := r.FindByName(ctx, userID, name)
	return p
}

type inMemoryTopicRepo struct {
	topics []*entities.Topic
}

func newInMemoryTopicRepo() *inMemoryTopicRepo {
	return &inMemoryTopicRepo{}
}

func (r *inMemoryTopicRepo) Create(ctx context.Context, topic *entities.Topic) (string, error) {
	r.topics = append(r.topics, topic)
	return topic.ID, nil
}

func (r *inMemoryTopicRepo) FindByID(ctx context.Context, topicID string) (*entities.Topic, error) {
	return nil, nil
}

func (r *inMemoryTopicRepo) ListByUserID(ctx context.Context, userID string) ([]*entities.Topic, error) {
	return r.topics, nil
}

func (r *inMemoryTopicRepo) FindRandomByUserID(ctx context.Context, userID string) (*entities.Topic, error) {
	if len(r.topics) == 0 {
		return nil, nil
	}
	return r.topics[0], nil
}

func (r *inMemoryTopicRepo) Update(ctx context.Context, topic *entities.Topic) error { return nil }
func (r *inMemoryTopicRepo) Delete(ctx context.Context, topicID string) error        { return nil }
func (r *inMemoryTopicRepo) FindByPrompt(ctx context.Context, userID string, promptName string) ([]*entities.Topic, error) {
	return nil, nil
}
func (r *inMemoryTopicRepo) FindByIdeasRange(ctx context.Context, userID string, minIdeas, maxIdeas int) ([]*entities.Topic, error) {
	return nil, nil
}

type stubUserRepo struct{}

func (s *stubUserRepo) Create(ctx context.Context, user *entities.User) (string, error) {
	return "", nil
}
func (s *stubUserRepo) FindByID(ctx context.Context, userID string) (*entities.User, error) {
	return &entities.User{ID: userID}, nil
}
func (s *stubUserRepo) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	return nil, nil
}
func (s *stubUserRepo) Update(ctx context.Context, userID string, updates map[string]interface{}) error {
	return nil
}
func (s *stubUserRepo) UpdateLinkedInToken(ctx context.Context, userID string, token string) error {
	return nil
}
func (s *stubUserRepo) Delete(ctx context.Context, userID string) error { return nil }

type stubIdeasRepo struct{}

func (s *stubIdeasRepo) CreateBatch(ctx context.Context, ideas []*entities.Idea) error { return nil }
func (s *stubIdeasRepo) ListByUserID(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
	return nil, nil
}
func (s *stubIdeasRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return 0, nil
}
func (s *stubIdeasRepo) ClearByUserID(ctx context.Context, userID string) error { return nil }

type stubLLM struct{}

func (s *stubLLM) SendRequest(ctx context.Context, prompt string) (string, error) { return "", nil }
func (s *stubLLM) GenerateIdeas(ctx context.Context, topic string, count int) ([]string, error) {
	return nil, nil
}
func (s *stubLLM) GenerateDrafts(ctx context.Context, idea string, userContext string) (interfaces.DraftSet, error) {
	return interfaces.DraftSet{}, nil
}
func (s *stubLLM) RefineDraft(ctx context.Context, draft string, userPrompt string, history []string) (string, error) {
	return "", nil
}
