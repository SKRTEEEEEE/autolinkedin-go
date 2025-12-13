package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/linkgen-ai/backend/src/domain/entities"
	infraServices "github.com/linkgen-ai/backend/src/infrastructure/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TDD Red test for Issue 3 Phase 0: ensure prompt loader skips legacy files
// and parses YAML front-matter correctly from seed/prompt directory.
func TestPromptLoader_SkipsOldFilesAndParsesFrontMatter(t *testing.T) {
	tmpDir := t.TempDir()

	validPrompt := `---
name: base1
type: ideas
---
Contenido para base1 con {ideas} ideas`

	legacyPrompt := `---
name: legacy
type: ideas
---
Debe ser ignorado`

	draftPrompt := `---
name: profesional
type: drafts
---
Plantilla de drafts`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "base1.idea.md"), []byte(validPrompt), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "legacy.old.md"), []byte(legacyPrompt), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "pro.draft.md"), []byte(draftPrompt), 0o644))

	loader := infraServices.NewPromptLoader(&noopLogger{})

	promptFiles, err := loader.LoadPromptsFromDir(tmpDir)
	require.NoError(t, err)

	// Should skip *.old.md legacy files and keep valid md entries
	require.Len(t, promptFiles, 2)

	names := []string{promptFiles[0].Name, promptFiles[1].Name}
	assert.Contains(t, names, "base1")
	assert.Contains(t, names, "profesional")

	prompts, err := loader.CreatePromptsFromFile("dev-user", promptFiles)
	require.NoError(t, err)
	require.Len(t, prompts, 2)

	assert.Equal(t, entities.PromptTypeIdeas, prompts[0].Type)
	assert.Equal(t, entities.PromptTypeDrafts, prompts[1].Type)
	assert.True(t, prompts[0].Active)
	assert.True(t, prompts[1].Active)
}

// noopLogger implements interfaces.Logger for tests
type noopLogger struct{}

func (l *noopLogger) Debug(msg string, fields ...interface{}) {}
func (l *noopLogger) Info(msg string, fields ...interface{})  {}
func (l *noopLogger) Warn(msg string, fields ...interface{})  {}
func (l *noopLogger) Error(msg string, fields ...interface{}) {}
