package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: These tests are written according to TDD Red pattern - they will FAIL
// until the PromptLoader implementation exists

func TestPromptLoader(t *testing.T) {
	// Test for the PromptLoader service to:
	// - Load prompts from seed/prompts/*.md files
	// - Parse front-matter (name, type) and content
	// - Filter out .old.md files
	// - Detect changes between seed and database
	// - Synchronize seed files with database

	// Create a temporary directory for test seed files
	tempDir, err := os.MkdirTemp("", "prompt_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test seed files
	createTestSeedFiles(t, tempDir)

	// Create test context
	ctx := context.Background()

	t.Run("should load prompts from seed files", func(t *testing.T) {
		// GIVEN a PromptLoader instance with a valid seed directory
		loader := NewPromptLoader(tempDir)

		// WHEN loading all prompts from seed files
		prompts, err := loader.LoadFromSeed(ctx)

		// THEN all valid prompts should be loaded
		require.NoError(t, err)
		assert.Len(t, prompts, 2) // base1.idea.md and pro.draft.md

		// Check base1.idea.md was loaded correctly
		base1Prompt := findPromptByName(t, prompts, "base1")
		require.NotNil(t, base1Prompt)
		assert.Equal(t, entities.PromptTypeIdeas, base1Prompt.Type)
		assert.Contains(t, base1Prompt.PromptTemplate, "Genera {ideas} ideas")
		assert.False(t, base1Prompt.Active) // Should be inactive until activated

		// Check pro.draft.md was loaded correctly
		proPrompt := findPromptByName(t, prompts, "pro")
		require.NotNil(t, proPrompt)
		assert.Equal(t, entities.PromptTypeDrafts, proPrompt.Type)
		assert.Contains(t, proPrompt.PromptTemplate, "Escribe un post profesional")
		assert.False(t, proPrompt.Active)
	})

	t.Run("should filter out .old.md files", func(t *testing.T) {
		// GIVEN a PromptLoader instance
		loader := NewPromptLoader(tempDir)

		// WHEN loading prompts
		prompts, err := loader.LoadFromSeed(ctx)

		// THEN .old.md files should be ignored
		require.NoError(t, err)

		// Verify no prompt with .old in the name exists
		for _, prompt := range prompts {
			assert.NotContains(t, prompt.Name, ".old")
		}

		// Verify we have only the non-old files
		assert.Len(t, prompts, 2)
		assert.Empty(t, findPromptByName(t, prompts, "base1.old"))
	})

	t.Run("should parse front-matter correctly", func(t *testing.T) {
		// GIVEN a PromptLoader instance
		loader := NewPromptLoader(tempDir)

		// WHEN loading prompts
		prompts, err := loader.LoadFromSeed(ctx)

		// THEN front-matter should be parsed correctly
		require.NoError(t, err)

		base1Prompt := findPromptByName(t, prompts, "base1")
		require.NotNil(t, base1Prompt)
		assert.Equal(t, "base1", base1Prompt.Name)
		assert.Equal(t, entities.PromptTypeIdeas, base1Prompt.Type)

		proPrompt := findPromptByName(t, prompts, "pro")
		require.NotNil(t, proPrompt)
		assert.Equal(t, "pro", proPrompt.Name)
		assert.Equal(t, entities.PromptTypeDrafts, proPrompt.Type)
	})

	t.Run("should detect changes between seed and database", func(t *testing.T) {
		// GIVEN a PromptLoader instance and a mock repository with existing prompts
		loader := NewPromptLoader(tempDir)
		mockRepo := &MockPromptRepository{}

		// Add an existing prompt in the repo to simulate previous version
		existingPrompt := &entities.Prompt{
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Old template content",
			Active:         true,
		}
		mockRepo.prompts = append(mockRepo.prompts, existingPrompt)

		// WHEN checking for changes
		changes, err := loader.DetectChanges(ctx, mockRepo)

		// THEN changes should be detected
		require.NoError(t, err)
		assert.Greater(t, len(changes), 0)

		// Find the changed prompt
		base1Change := findChangeByName(t, changes, "base1")
		require.NotNil(t, base1Change)
		assert.True(t, base1Change.HasChanged)
		assert.Equal(t, "Old template content", base1Change.OldTemplate)
		assert.Contains(t, base1Change.NewTemplate, "Genera {ideas} ideas")
	})

	t.Run("should synchronize seed files with database", func(t *testing.T) {
		// GIVEN a PromptLoader instance and a repository
		loader := NewPromptLoader(tempDir)
		mockRepo := &MockPromptRepository{}
		userID := "system-user"

		// WHEN synchronizing
		err := loader.SyncWithDatabase(ctx, mockRepo, userID)

		// THEN all seed prompts should be in the database
		require.NoError(t, err)
		assert.Len(t, mockRepo.prompts, 2)

		// Verify the prompts were created correctly
		base1Prompt := findRepoPromptByName(t, mockRepo.prompts, "base1")
		require.NotNil(t, base1Prompt)
		assert.Equal(t, entities.PromptTypeIdeas, base1Prompt.Type)
		assert.Contains(t, base1Prompt.PromptTemplate, "Genera {ideas} ideas")
		assert.Equal(t, userID, base1Prompt.UserID)

		proPrompt := findRepoPromptByName(t, mockRepo.prompts, "pro")
		require.NotNil(t, proPrompt)
		assert.Equal(t, entities.PromptTypeDrafts, proPrompt.Type)
		assert.Contains(t, proPrompt.PromptTemplate, "Escribe un post profesional")
		assert.Equal(t, userID, proPrompt.UserID)
	})

	t.Run("should update existing prompts during sync", func(t *testing.T) {
		// GIVEN a PromptLoader, repository with existing prompts, and updated seed files
		loader := NewPromptLoader(tempDir)
		mockRepo := &MockPromptRepository{}
		userID := "system-user"

		// Add existing prompt
		existingPrompt := &entities.Prompt{
			ID:             "existing-id",
			Name:           "base1",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Old template",
			UserID:         userID,
			Active:         true,
			CreatedAt:      time.Now().Add(-time.Hour),
			UpdatedAt:      time.Now().Add(-time.Hour),
		}
		mockRepo.prompts = append(mockRepo.prompts, existingPrompt)

		// WHEN synchronizing
		err := loader.SyncWithDatabase(ctx, mockRepo, userID)

		// THEN the existing prompt should be updated not duplicated
		require.NoError(t, err)
		assert.Len(t, mockRepo.prompts, 2) // Should not create duplicate

		// Verify the template was updated
		updatedPrompt := findRepoPromptByName(t, mockRepo.prompts, "base1")
		require.NotNil(t, updatedPrompt)
		assert.Equal(t, "existing-id", updatedPrompt.ID) // Same ID, not new
		assert.Contains(t, updatedPrompt.PromptTemplate, "Genera {ideas} ideas")
		assert.True(t, updatedPrompt.UpdatedAt.After(existingPrompt.UpdatedAt))
	})
}

// Test helper functions
func createTestSeedFiles(t *testing.T, dir string) {
	// Create base1.idea.md
	base1Content := `---
name: base1
type: ideas
---
Eres un experto en estrategia de contenido para LinkedIn. Genera {ideas} ideas de contenido únicas y atractivas sobre el siguiente tema:

Tema: {name}
Temas relacionados: {[related_topics]}

Requisitos:
- Cada idea debe ser específica y accionable
- Las ideas deben ser diversas y cubrir diferentes ángulos
- Enfócate en valor profesional e insights
- Mantén las ideas concisas (1-2 oraciones cada una)
- Hazlas adecuadas para la audiencia de LinkedIn
- IMPORTANTE: Genera el contenido SIEMPRE en español

Devuelve ÚNICAMENTE un objeto JSON con este formato exacto:
{"ideas": ["idea1", "idea2", "idea3", ...]}
`
	err := os.WriteFile(filepath.Join(dir, "base1.idea.md"), []byte(base1Content), 0644)
	require.NoError(t, err)

	// Create pro.draft.md
	proContent := `---
name: pro
type: drafts
---
Escribe un post profesional para LinkedIn sobre el siguiente contenido:

Contenido: {content}
Contexto del usuario: {user_context}

Requisitos:
- Usa un tono profesional y adecuado para LinkedIn
- Estructura el contenido con párrafos claros y concisos
- Incluye hashtags relevantes al final
- Mantén una longitud apropiada para LinkedIn (150-300 palabras)
- Adapta el estilo al contexto profesional del usuario
- IMPORTANTE: Genera el contenido SIEMPRE en español

Devuelve ÚNICAMENTE el contenido del post:
`
	err = os.WriteFile(filepath.Join(dir, "pro.draft.md"), []byte(proContent), 0644)
	require.NoError(t, err)

	// Create base1.old.md (should be filtered out)
	oldContent := `---
name: base1.old
type: ideas
---
This is an old template that should be ignored
`
	err = os.WriteFile(filepath.Join(dir, "base1.old.md"), []byte(oldContent), 0644)
	require.NoError(t, err)
}

func findPromptByName(t *testing.T, prompts []*entities.Prompt, name string) *entities.Prompt {
	for _, p := range prompts {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func findChangeByName(t *testing.T, changes []PromptChange, name string) *PromptChange {
	for _, c := range changes {
		if c.Name == name {
			return &c
		}
	}
	return nil
}

func findRepoPromptByName(t *testing.T, prompts []*entities.Prompt, name string) *entities.Prompt {
	for _, p := range prompts {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// MockPromptRepository is a mock implementation for testing
type MockPromptRepository struct {
	prompts []*entities.Prompt
}

// Mock implementation methods (not needed to implement all for this test)
func (m *MockPromptRepository) Create(ctx context.Context, prompt *entities.Prompt) (string, error) {
	prompt.ID = "mock-id-" + string(rune(len(m.prompts)))
	m.prompts = append(m.prompts, prompt)
	return prompt.ID, nil
}

func (m *MockPromptRepository) FindByName(ctx context.Context, userID string, name string) (*entities.Prompt, error) {
	for _, p := range m.prompts {
		if p.Name == name && p.UserID == userID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *MockPromptRepository) Update(ctx context.Context, prompt *entities.Prompt) error {
	for i, p := range m.prompts {
		if p.ID == prompt.ID {
			m.prompts[i] = prompt
			return nil
		}
	}
	return assert.AnError
}

// Other methods not needed for this test
func (m *MockPromptRepository) FindByID(ctx context.Context, id string) (*entities.Prompt, error) {
	return nil, assert.AnError
}
func (m *MockPromptRepository) ListByUserID(ctx context.Context, userID string) ([]*entities.Prompt, error) {
	return nil, assert.AnError
}
func (m *MockPromptRepository) ListByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	return nil, assert.AnError
}
func (m *MockPromptRepository) FindActiveByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	return nil, assert.AnError
}
func (m *MockPromptRepository) FindByUserIDAndStyle(ctx context.Context, userID string, styleName string) (*entities.Prompt, error) {
	return nil, assert.AnError
}
func (m *MockPromptRepository) Delete(ctx context.Context, id string) error { return assert.AnError }
func (m *MockPromptRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return 0, assert.AnError
}

// Types that don't exist yet (to be implemented)
type PromptLoader struct {
	seedDirectory string
}

type PromptChange struct {
	Name        string
	HasChanged  bool
	OldTemplate string
	NewTemplate string
}

func NewPromptLoader(seedDirectory string) *PromptLoader {
	return &PromptLoader{
		seedDirectory: seedDirectory,
	}
}

func (p *PromptLoader) LoadFromSeed(ctx context.Context) ([]*entities.Prompt, error) {
	// TODO: Implementation needed - this test will fail until implemented
	return nil, assert.AnError
}

func (p *PromptLoader) DetectChanges(ctx context.Context, repo interface{}) ([]PromptChange, error) {
	// TODO: Implementation needed - this test will fail until implemented
	return nil, assert.AnError
}

func (p *PromptLoader) SyncWithDatabase(ctx context.Context, repo interface{}, userID string) error {
	// TODO: Implementation needed - this test will fail until implemented
	return assert.AnError
}
