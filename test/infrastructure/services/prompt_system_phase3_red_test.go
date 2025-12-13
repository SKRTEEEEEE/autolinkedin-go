package services_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/infrastructure/services"
)

// TDD Red tests for Issue 3 (3-3 Implementación del Sistema de Prompts)
// These assert the expected prompt loader sync flow and prompt engine behaviour.

func TestPromptLoader_SyncSeedPromptsAndDetectsChanges(t *testing.T) {
	ctx := context.Background()
	promptDir := t.TempDir()

	writePromptFile(t, promptDir, "base1.idea.md", `---
name: base1
type: ideas
---
Genera {ideas} ideas para {name} con {[related_topics]}`)

	writePromptFile(t, promptDir, "pro.draft.md", `---
name: pro
type: drafts
---
Escribe un post sobre {content} usando {user_context}`)

	// Should be ignored
	writePromptFile(t, promptDir, "legacy.old.md", `---
name: legacy
type: ideas
---
Contenido obsoleto`)

	loader := services.NewPromptLoader(&noopLogger{})

	repo := newFakePromptsRepo()
	repo.prompts = append(repo.prompts, &entities.Prompt{
		ID:             "existing-base1",
		UserID:         "dev-user",
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "plantilla vieja {name}",
		Active:         false,
		CreatedAt:      time.Now().Add(-2 * time.Hour),
		UpdatedAt:      time.Now().Add(-2 * time.Hour),
	})

	syncedPrompts, err := loader.SyncSeedPrompts(ctx, "dev-user", promptDir, repo)
	require.NoError(t, err)

	// Should upsert prompts from seed files and ignore *.old.md
	require.Len(t, syncedPrompts, 2)
	assert.Equal(t, 2, repo.countByUser("dev-user"))

	base1 := repo.findByName("dev-user", "base1")
	require.NotNil(t, base1)
	assert.True(t, base1.Active)
	assert.Contains(t, base1.PromptTemplate, "Genera {ideas}")
	assert.True(t, base1.UpdatedAt.After(base1.CreatedAt))

	pro := repo.findByName("dev-user", "pro")
	require.NotNil(t, pro)
	assert.Equal(t, entities.PromptTypeDrafts, pro.Type)
	assert.Contains(t, pro.PromptTemplate, "{content}")

	// Legacy prompt must not be stored
	assert.Nil(t, repo.findByName("dev-user", "legacy"))
}

func TestPromptEngine_ReplacesVariablesAndCaches(t *testing.T) {
	ctx := context.Background()
	repo := newFakePromptsRepo()

	now := time.Now()
	repo.prompts = append(repo.prompts,
		&entities.Prompt{
			ID:             "p-base1",
			UserID:         "user-1",
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Genera {ideas} ideas sobre {name} | Temas: {[related_topics]}",
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		&entities.Prompt{
			ID:             "p-pro",
			UserID:         "user-1",
			Name:           "pro",
			Type:           entities.PromptTypeDrafts,
			PromptTemplate: "Post: {content}\nContexto: {user_context}",
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	)

	engine := services.NewPromptEngine(repo, &noopLogger{})

	topic := &entities.Topic{
		ID:            "topic-1",
		UserID:        "user-1",
		Name:          "Go Concurrency",
		Ideas:         3,
		RelatedTopics: []string{"goroutines", "channels", "mutexes"},
		Active:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	user := &entities.User{
		ID:    "user-1",
		Email: "dev@example.com",
		Configuration: map[string]interface{}{
			"name":            "Jane Doe",
			"expertise":       "Backend Engineering",
			"tone_preference": "Professional",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	idea := &entities.Idea{
		ID:        "idea-1",
		UserID:    "user-1",
		TopicID:   topic.ID,
		Content:   "Aprender patrones de concurrencia en Go",
		Selected:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	ideasPrompt, err := engine.ProcessPrompt(ctx, user.ID, "base1", entities.PromptTypeIdeas, topic, nil, user)
	require.NoError(t, err)
	assert.NotContains(t, ideasPrompt, "{ideas}")
	assert.Contains(t, ideasPrompt, "3 ideas")
	assert.Contains(t, ideasPrompt, "Go Concurrency")
	assert.Contains(t, ideasPrompt, "goroutines, channels, mutexes")

	draftsPrompt, err := engine.ProcessPrompt(ctx, user.ID, "pro", entities.PromptTypeDrafts, nil, idea, user)
	require.NoError(t, err)
	assert.NotContains(t, draftsPrompt, "{content}")
	assert.NotContains(t, draftsPrompt, "{user_context}")
	assert.Contains(t, draftsPrompt, idea.Content)
	assert.Contains(t, draftsPrompt, "Jane Doe")
	assert.Contains(t, draftsPrompt, "Backend Engineering")

	// Cache should store processed prompts to avoid reprocessing
	cached := engine.CacheSize()
	assert.GreaterOrEqual(t, cached, 2)
}

// Helpers

type noopLogger struct{}

func (n *noopLogger) Debug(msg string, fields ...interface{}) {}
func (n *noopLogger) Info(msg string, fields ...interface{})  {}
func (n *noopLogger) Warn(msg string, fields ...interface{})  {}
func (n *noopLogger) Error(msg string, fields ...interface{}) {}

type fakePromptsRepo struct {
	prompts []*entities.Prompt
}

func newFakePromptsRepo() *fakePromptsRepo {
	return &fakePromptsRepo{prompts: []*entities.Prompt{}}
}

func (r *fakePromptsRepo) Create(ctx context.Context, prompt *entities.Prompt) (string, error) {
	r.prompts = append(r.prompts, prompt)
	return prompt.ID, nil
}

func (r *fakePromptsRepo) FindByID(ctx context.Context, id string) (*entities.Prompt, error) {
	for _, p := range r.prompts {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, nil
}

func (r *fakePromptsRepo) FindByName(ctx context.Context, userID string, name string) (*entities.Prompt, error) {
	return r.findByName(userID, name), nil
}

func (r *fakePromptsRepo) ListByUserID(ctx context.Context, userID string) ([]*entities.Prompt, error) {
	var out []*entities.Prompt
	for _, p := range r.prompts {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakePromptsRepo) ListByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	var out []*entities.Prompt
	for _, p := range r.prompts {
		if p.UserID == userID && p.Type == promptType {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakePromptsRepo) FindActiveByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	var out []*entities.Prompt
	for _, p := range r.prompts {
		if p.UserID == userID && p.Type == promptType && p.Active {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakePromptsRepo) FindByUserIDAndStyle(ctx context.Context, userID string, styleName string) (*entities.Prompt, error) {
	for _, p := range r.prompts {
		if p.UserID == userID && p.StyleName == styleName && p.Active {
			return p, nil
		}
	}
	return nil, nil
}

func (r *fakePromptsRepo) Update(ctx context.Context, prompt *entities.Prompt) error {
	for i, p := range r.prompts {
		if p.ID == prompt.ID {
			r.prompts[i] = prompt
			return nil
		}
	}
	r.prompts = append(r.prompts, prompt)
	return nil
}

func (r *fakePromptsRepo) Delete(ctx context.Context, id string) error { return nil }

func (r *fakePromptsRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	for _, p := range r.prompts {
		if p.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *fakePromptsRepo) findByName(userID, name string) *entities.Prompt {
	for _, p := range r.prompts {
		if p.UserID == userID && p.Name == name {
			return p
		}
	}
	return nil
}

func (r *fakePromptsRepo) countByUser(userID string) int {
	count := 0
	for _, p := range r.prompts {
		if p.UserID == userID {
			count++
		}
	}
	return count
}

func writePromptFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644)
	require.NoError(t, err)
}

var _ interfaces.PromptsRepository = (*fakePromptsRepo)(nil)
var _ interfaces.Logger = (*noopLogger)(nil)
