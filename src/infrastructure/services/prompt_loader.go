package services

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/yaml.v3"
)

// PromptLoader handles loading prompts from markdown files with front-matter
type PromptLoader struct {
	logger interfaces.Logger
}

// NewPromptLoader creates a new PromptLoader instance
func NewPromptLoader(logger interfaces.Logger) *PromptLoader {
	return &PromptLoader{
		logger: logger,
	}
}

// PromptFile represents a parsed prompt file
type PromptFile struct {
	Name           string
	Type           string
	PromptTemplate string
}

// LoadPromptsFromDir loads all prompt files from a directory
func (pl *PromptLoader) LoadPromptsFromDir(dirPath string) ([]*PromptFile, error) {
	var promptFiles []*PromptFile

	// Read all .md files in the directory
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .md files
		if filepath.Ext(path) != ".md" {
			return nil
		}

		// Skip legacy prompt files
		if strings.HasSuffix(path, ".old.md") {
			pl.logger.Info("Skipping legacy prompt file", "path", path)
			return nil
		}

		// Parse the prompt file
		promptFile, err := pl.parsePromptFile(path)
		if err != nil {
			pl.logger.Warn("Failed to parse prompt file", "path", path, "error", err)
			return nil // Don't fail the entire process for one file
		}

		promptFiles = append(promptFiles, promptFile)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	return promptFiles, nil
}

// parsePromptFile parses a markdown file with YAML front-matter
func (pl *PromptLoader) parsePromptFile(filePath string) (*PromptFile, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Split content by front-matter separator
	parts := strings.Split(string(content), "---")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid front-matter format in %s", filePath)
	}

	// Parse YAML front-matter
	var meta struct {
		Name string `yaml:"name"`
		Type string `yaml:"type"`
	}

	if err := yaml.Unmarshal([]byte(parts[1]), &meta); err != nil {
		return nil, fmt.Errorf("invalid front-matter in %s: %w", filePath, err)
	}

	name := strings.TrimSpace(meta.Name)
	promptType := strings.TrimSpace(meta.Type)

	if name == "" {
		return nil, fmt.Errorf("missing name in front-matter of %s", filePath)
	}

	if promptType == "" {
		return nil, fmt.Errorf("missing type in front-matter of %s", filePath)
	}

	if promptType != string(entities.PromptTypeIdeas) && promptType != string(entities.PromptTypeDrafts) {
		return nil, fmt.Errorf("invalid prompt type %s in %s", promptType, filePath)
	}

	// Get the template content (everything after the second ---)
	templateContent := strings.TrimSpace(strings.Join(parts[2:], "---"))
	if templateContent == "" {
		return nil, fmt.Errorf("empty template content in %s", filePath)
	}

	return &PromptFile{
		Name:           name,
		Type:           promptType,
		PromptTemplate: templateContent,
	}, nil
}

// CreatePromptsFromFile creates prompt entities from loaded files
func (pl *PromptLoader) CreatePromptsFromFile(userID string, promptFiles []*PromptFile) ([]*entities.Prompt, error) {
	now := time.Now()
	var prompts []*entities.Prompt

	for _, promptFile := range promptFiles {
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptType(promptFile.Type),
			Name:           promptFile.Name,
			StyleName:      promptFile.Name, // For backward compatibility
			PromptTemplate: promptFile.PromptTemplate,
			Active:         true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		// Validate the prompt
		if err := prompt.Validate(); err != nil {
			pl.logger.Warn("Invalid prompt, skipping", "name", promptFile.Name, "error", err)
			continue
		}

		prompts = append(prompts, prompt)
	}

	return prompts, nil
}
