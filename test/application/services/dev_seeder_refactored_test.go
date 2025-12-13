package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestDevSeederRefactored(t *testing.T) {
	// Test for the new DevSeeder functionality:
	// - Read prompts from seed/prompt/*.md files
	// - Parse front-matter YAML (name, type, content)
	// - Seed topics with prompt references
	// - Synchronize file changes to database

	// Setup test database
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(context.Background())

	// Use a test database
	testDB := client.Database("test_dev_seeder_refactored")
	defer testDB.Drop(context.Background())

	// Setup repositories
	promptsCollection := testDB.Collection("prompts")
	topicsCollection := testDB.Collection("topics")
	ideasCollection := testDB.Collection("ideas")

	promptsRepo := repositories.NewPromptsRepository(promptsCollection)
	topicsRepo := repositories.NewTopicRepository(topicsCollection)
	ideasRepo := repositories.NewIdeasRepository(ideasCollection)

	// Create seeder with mock file system
	seeder := NewDevSeederRefactored(promptsRepo, topicsRepo, ideasRepo)

	t.Run("should parse prompt files with front-matter correctly", func(t *testing.T) {
		// GIVEN a mock prompt file with valid front-matter
		fileContent := `---
name: base1
type: ideas
---
Eres un experto en estrategia de contenido para LinkedIn. Genera {ideas} ideas de contenido únicas y atractivas sobre el siguiente tema:

Tema: {name}

Requisitos:
- Cada idea debe ser específica y accionable
- No más de {ideas}

Devuelve ÚNICAMENTE un objeto JSON con este formato exacto:
{"ideas": ["idea1", "idea2", "idea3"]}`

		// WHEN parsing the file content
		prompt, err := seeder.ParsePromptFile("test.md", []byte(fileContent))

		// THEN the prompt should be correctly parsed
		require.NoError(t, err)
		assert.Equal(t, "base1", prompt.Name)
		assert.Equal(t, entities.PromptTypeIdeas, prompt.Type)
		assert.Contains(t, prompt.PromptTemplate, "Genera {ideas} ideas")
		assert.True(t, prompt.Active)
		assert.Equal(t, DevUserID, prompt.UserID)
	})

	t.Run("should handle draft type prompt files", func(t *testing.T) {
		// GIVEN a draft type prompt file
		fileContent := `---
name: profesional
type: drafts
---
Eres un experto creador de contenido para LinkedIn.

Basándote en la siguiente idea:
{content}

Instrucciones clave:
- Escribe SIEMPRE en español neutro profesional.
- Cada post debe tener 120-260 palabras

FORMATO OBLIGATORIO: Responde ÚNICAMENTE con el JSON siguiente:
{
  "posts": ["Post 1", "Post 2"],
  "articles": ["Título\\n\\nCuerpo del artículo"]
}`

		// WHEN parsing the file content
		prompt, err := seeder.ParsePromptFile("professional.md", []byte(fileContent))

		// THEN the prompt should be correctly parsed
		require.NoError(t, err)
		assert.Equal(t, "profesional", prompt.Name)
		assert.Equal(t, entities.PromptTypeDrafts, prompt.Type)
		assert.Contains(t, prompt.PromptTemplate, "Basándote en la siguiente idea")
		assert.True(t, prompt.Active)
	})

	t.Run("should fail to parse invalid front-matter", func(t *testing.T) {
		// GIVEN a file with invalid front-matter
		invalidContent := `---
name: base1
type: invalid_type
---
Este es un contenido sin front-matter válido.`

		// WHEN parsing the file content
		prompt, err := seeder.ParsePromptFile("invalid.md", []byte(invalidContent))

		// THEN it should fail with appropriate error
		require.Error(t, err)
		assert.Nil(t, prompt)
		assert.Contains(t, err.Error(), "invalid prompt type")
	})

	t.Run("should fail to parse missing front-matter", func(t *testing.T) {
		// GIVEN a file without front-matter
		missingContent := `Este es un archivo sin front-matter.
Solo tiene el contenido del template.`

		// WHEN parsing the file content
		prompt, err := seeder.ParsePromptFile("missing.md", []byte(missingContent))

		// THEN it should fail
		require.Error(t, err)
		assert.Nil(t, prompt)
		assert.Contains(t, err.Error(), "front-matter")
	})

	t.Run("should seed prompts from file system", func(t *testing.T) {
		// GIVEN mocked file system with prompt files
		mockFS := NewMockFileSystem()
		mockFS.AddFile("base1.idea.md", []byte(`---
name: base1
type: ideas
---
Genera {ideas} ideas sobre {name}`))

		mockFS.AddFile("profesional.draft.md", []byte(`---
name: profesional
type: drafts
---
Crea posts sobre {content}`))

		// WHEN seeding prompts from the file system
		err := seeder.SeedPromptsFromFiles(context.Background(), mockFS)

		// THEN prompts should be created in the database
		require.NoError(t, err)

		// AND the base1 prompt should exist
		base1Prompt, err := promptsRepo.FindByName(context.Background(), DevUserID, "base1")
		require.NoError(t, err)
		assert.NotNil(t, base1Prompt)
		assert.Equal(t, entities.PromptTypeIdeas, base1Prompt.Type)

		// AND the profesional prompt should exist
		profPrompt, err := promptsRepo.FindByName(context.Background(), DevUserID, "profesional")
		require.NoError(t, err)
		assert.NotNil(t, profPrompt)
		assert.Equal(t, entities.PromptTypeDrafts, profPrompt.Type)
	})

	t.Run("should update existing prompts when files change", func(t *testing.T) {
		// GIVEN an existing prompt in the database
		existingPrompt := &entities.Prompt{
			Name:           "base1",
			UserID:         DevUserID,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Old template",
			Active:         true,
			CreatedAt:      time.Now().Add(-1 * time.Hour),
			UpdatedAt:      time.Now().Add(-1 * time.Hour),
		}

		_, err := promptsRepo.Create(context.Background(), existingPrompt)
		require.NoError(t, err)

		// AND a modified version in the file system
		mockFS := NewMockFileSystem()
		mockFS.AddFile("base1.idea.md", []byte(`---
name: base1
type: ideas
---
New and improved template with {ideas} ideas for {name}`))

		// WHEN seeding prompts (which should update existing ones)
		err = seeder.SeedPromptsFromFiles(context.Background(), mockFS)

		// THEN the prompt should be updated not duplicated
		require.NoError(t, err)

		// AND there should still be only one base1 prompt
		base1Prompts, err := promptsRepo.FindByNameAllUsers(context.Background(), "base1")
		require.NoError(t, err)
		assert.Len(t, base1Prompts, 1)

		// AND the template should be updated
		updatedPrompt := base1Prompts[0]
		assert.Equal(t, "New and improved template", updatedPrompt.PromptTemplate[:28]) // Check beginning
		assert.True(t, updatedPrompt.UpdatedAt.After(existingPrompt.UpdatedAt))
	})

	t.Run("should seed topics with prompt references", func(t *testing.T) {
		// GIVEN prompts are already seeded
		mockFS := NewMockFileSystem()
		mockFS.AddFile("base1.idea.md", []byte(`---
name: base1
type: ideas
---
Template for ideas`))

		err := seeder.SeedPromptsFromFiles(context.Background(), mockFS)
		require.NoError(t, err)

		// WHEN seeding default topics
		err = seeder.SeedDefaultTopics(context.Background())

		// THEN default topics should be created with prompt references
		require.NoError(t, err)

		// AND each topic should have a prompt reference
		topics, err := topicsRepo.ListByUserID(context.Background(), DevUserID)
		require.NoError(t, err)
		assert.NotEmpty(t, topics)

		for _, topic := range topics {
			assert.NotEmpty(t, topic.Prompt, "Topic should have a prompt reference")
			// Verify the referenced prompt exists
			referencedPrompt, err := promptsRepo.FindByName(context.Background(), DevUserID, topic.Prompt)
			assert.NoError(t, err)
			assert.NotNil(t, referencedPrompt)
		}
	})

	t.Run("should create topic with correct ideas count based on topic configuration", func(t *testing.T) {
		// GIVEN prompts are seeded
		mockFS := NewMockFileSystem()
		mockFS.AddFile("base1.idea.md", []byte(`---
name: base1
type: ideas
---
Template for idea generation`))

		err := seeder.SeedPromptsFromFiles(context.Background(), mockFS)
		require.NoError(t, err)

		// WHEN creating a topic with custom ideas count
		topicConfig := TopicConfig{
			Name:        "Custom Ideas Topic",
			Description: "Topic with custom ideas count",
			Category:    "Test",
			Priority:    7,
			Ideas:       5, // Custom ideas count
			Prompt:      "base1", // Using seeded prompt
			Active:      true,
		}

		createdTopic, err := seeder.CreateTopicWithConfig(context.Background(), topicConfig)
		require.NoError(t, err)
		assert.NotNil(t, createdTopic)

		// THEN the topic should have the correct ideas count
		assert.Equal(t, 5, createdTopic.Ideas)
		assert.Equal(t, "base1", createdTopic.Prompt)
	})

	t.Run("should sync topic ideas count with prompt template variables", func(t *testing.T) {
		// GIVEN a prompt with {ideas} variable
		mockFS := NewMockFileSystem()
		mockFS.AddFile("dynamic.idea.md", []byte(`---
name: dynamic
type: ideas
---
Generate exactly {ideas} ideas about {name} with their descriptions`))

		// Seed the prompt
		err := seeder.SeedPromptsFromFiles(context.Background(), mockFS)
		require.NoError(t, err)

		// WHEN creating a topic that uses this prompt
		topicConfig := TopicConfig{
			Name:        "Dynamic Topic",
			Description: "Topic with dynamic ideas count",
			Ideas:       7,
			Prompt:      "dynamic",
			Active:      true,
		}

		topic, err := seeder.CreateTopicWithConfig(context.Background(), topicConfig)
		require.NoError(t, err)

		// THEN the topic should have the correct ideas count that matches the prompt expectation
		assert.Equal(t, 7, topic.Ideas)
		assert.Equal(t, "dynamic", topic.Prompt)

		// AND the prompt should contain the ideas variable
		referencedPrompt, err := promptsRepo.FindByName(context.Background(), DevUserID, "dynamic")
		require.NoError(t, err)
		assert.Contains(t, referencedPrompt.PromptTemplate, "{ideas}")
	})

	t.Run("should complete full seeding process", func(t *testing.T) {
		// GIVEN a complete mock file system with prompts
		mockFS := NewMockFileSystem()
		mockFS.AddFile("base1.idea.md", []byte(`---
name: base1
type: ideas
---
Genera {ideas} ideas sobre {name}`))

		mockFS.AddFile("profesional.draft.md", []byte(`---
name: profesional
type: drafts
---
Crea posts profesionales sobre {content}`))

		// Clear database
		_, err := promptsCollection.DeleteMany(context.Background(), bson.M{})
		require.NoError(t, err)
		_, err = topicsCollection.DeleteMany(context.Background(), bson.M{})
		require.NoError(t, err)

		// WHEN running the complete seeding process
		err = seeder.SeedAll(context.Background(), mockFS)
		require.NoError(t, err)

		// THEN all system components should be properly seeded
		// Check prompts
		prompts, err := promptsRepo.ListByUserID(context.Background(), DevUserID)
		require.NoError(t, err)
		assert.Len(t, prompts, 2) // base1 and profesional

		// Check topics
		topics, err := topicsRepo.ListByUserID(context.Background(), DevUserID)
		require.NoError(t, err)
		assert.NotEmpty(t, topics)

		// Check topic-prompt relationships
		for _, topic := range topics {
			assert.NotEmpty(t, topic.Prompt)
			referencedPrompt, err := promptsRepo.FindByName(context.Background(), DevUserID, topic.Prompt)
			assert.NoError(t, err)
			assert.NotNil(t, referencedPrompt)
		}
	})
}

// Constants
const (
	DevUserID = "dev-user-00000000-0000-0000-0000-000000000000"
)

// MockFileSystem simulates a file system for testing
type MockFileSystem struct {
	files map[string][]byte
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
	}
}

func (m *MockFileSystem) AddFile(path string, content []byte) {
	m.files[path] = content
}

func (m *MockFileSystem) ReadDir(dirname string) ([]string, error) {
	var files []string
	for path := range m.files {
		files = append(files, path)
	}
	return files, nil
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	content, exists := m.files[filename]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", filename)
	}
	return content, nil
}

// TopicConfig represents the configuration for creating a topic
type TopicConfig struct {
	Name        string
	Description string
	Category    string
	Priority    int
	Ideas       int
	Prompt      string
	Active      bool
}

// DevSeederRefactored represents the refactored version of the development seeder
type DevSeederRefactored struct {
	promptsRepo repositories.PromptsRepository
	topicsRepo  repositories.TopicRepository
	ideasRepo   repositories.IdeasRepository
}

func NewDevSeederRefactored(
	promptsRepo repositories.PromptsRepository,
	topicsRepo repositories.TopicRepository,
	ideasRepo repositories.IdeasRepository,
) *DevSeederRefactored {
	return &DevSeederRefactored{
		promptsRepo: promptsRepo,
		topicsRepo:  topicsRepo,
		ideasRepo:   ideasRepo,
	}
}

// ParsePromptFile parses a prompt file with front-matter
func (s *DevSeederRefactored) ParsePromptFile(filename string, content []byte) (*entities.Prompt, error) {
	// This is a mock implementation - in the real implementation,
	// this would parse YAML front-matter and extract prompt content
	
	// For testing purposes, we'll simulate parsing
	lines := strings.Split(string(content), "\n")
	
	var name, ptype, template string
	inContent := false
	var contentLines []string
	
	for _, line := range lines {
		if strings.HasPrefix(line, "---") {
			if !inContent {
				inContent = true
				continue
			} else {
				// Front-matter done, start content
				inContent = false
				continue
			}
		}
		
		if inContent {
			if strings.HasPrefix(line, "name:") {
				name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			} else if strings.HasPrefix(line, "type:") {
				ptype = strings.TrimSpace(strings.TrimPrefix(line, "type:"))
			}
		} else {
			// This is content, not front-matter
			contentLines = append(contentLines, line)
		}
	}
	
	template = strings.Join(contentLines, "\n")
	template = strings.TrimSpace(template)
	
	// Validate
	if name == "" {
		return nil, fmt.Errorf("prompt name is required")
	}
	
	var promptType entities.PromptType
	switch ptype {
	case "ideas":
		promptType = entities.PromptTypeIdeas
	case "drafts":
		promptType = entities.PromptTypeDrafts
	default:
		return nil, fmt.Errorf("invalid prompt type: %s", ptype)
	}
	
	if template == "" {
		return nil, fmt.Errorf("prompt template cannot be empty")
	}
	
	return &entities.Prompt{
		Name:           name,
		UserID:         DevUserID,
		Type:           promptType,
		PromptTemplate: template,
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

// SeedPromptsFromFiles seeds prompts from the file system
func (s *DevSeederRefactored) SeedPromptsFromFiles(ctx context.Context, fs FileSystem) error {
	files, err := fs.ReadDir("seed/prompt")
	if err != nil {
		return fmt.Errorf("failed to read prompt directory: %w", err)
	}
	
	for _, file := range files {
		if !strings.HasSuffix(file, ".md") {
			continue
		}
		
		content, err := fs.ReadFile("seed/prompt/" + file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}
		
		prompt, err := s.ParsePromptFile(file, content)
		if err != nil {
			return fmt.Errorf("failed to parse file %s: %w", file, err)
		}
		
		// Check if prompt already exists
		existingPrompt, err := s.promptsRepo.FindByName(ctx, DevUserID, prompt.Name)
		if err != nil {
			return fmt.Errorf("failed to check existing prompt: %w", err)
		}
		
		if existingPrompt != nil {
			// Update existing prompt
			existingPrompt.PromptTemplate = prompt.PromptTemplate
			existingPrompt.Active = prompt.Active
			existingPrompt.UpdatedAt = time.Now()
			err = s.promptsRepo.Update(ctx, existingPrompt)
			if err != nil {
				return fmt.Errorf("failed to update prompt %s: %w", prompt.Name, err)
			}
		} else {
			// Create new prompt
			_, err = s.promptsRepo.Create(ctx, prompt)
			if err != nil {
				return fmt.Errorf("failed to create prompt %s: %w", prompt.Name, err)
			}
		}
	}
	
	return nil
}

// SeedDefaultTopics seeds default development topics
func (s *DevSeederRefactored) SeedDefaultTopics(ctx context.Context) error {
	defaultTopics := []TopicConfig{
		{
			Name:        "Marketing Digital",
			Description: "Contenido sobre estrategias de marketing digital",
			Category:    "Marketing",
			Priority:    7,
			Ideas:       3,
			Prompt:      "base1",
			Active:      true,
		},
		{
			Name:        "Liderazgo",
			Description: "Contenido sobre liderazgo empresarial",
			Category:    "Management",
			Priority:    8,
			Ideas:       2,
			Prompt:      "base1",
			Active:      true,
		},
	}
	
	for _, config := range defaultTopics {
		_, err := s.CreateTopicWithConfig(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to create topic %s: %w", config.Name, err)
		}
	}
	
	return nil
}

// CreateTopicWithConfig creates a topic with the given configuration
func (s *DevSeederRefactored) CreateTopicWithConfig(ctx context.Context, config TopicConfig) (*entities.Topic, error) {
	// Validate that the referenced prompt exists
	referencedPrompt, err := s.promptsRepo.FindByName(ctx, DevUserID, config.Prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to validate prompt reference: %w", err)
	}
	if referencedPrompt == nil {
		return nil, fmt.Errorf("referenced prompt not found: %s", config.Prompt)
	}
	
	topic := &entities.Topic{
		UserID:    DevUserID,
		Name:      config.Name,
		Ideas:     config.Ideas,
		Prompt:    config.Prompt,
		Active:    config.Active,
		CreatedAt: time.Now(),
	}
	
	id, err := s.topicsRepo.Create(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}
	
	topic.ID = id
	return topic, nil
}

// SeedAll runs the complete seeding process
func (s *DevSeederRefactored) SeedAll(ctx context.Context, fs FileSystem) error {
	// Seed prompts first
	if err := s.SeedPromptsFromFiles(ctx, fs); err != nil {
		return fmt.Errorf("failed to seed prompts: %w", err)
	}
	
	// Then seed topics that reference the prompts
	if err := s.SeedDefaultTopics(ctx); err != nil {
		return fmt.Errorf("failed to seed topics: %w", err)
	}
	
	return nil
}

// FileSystem interface for file operations
type FileSystem interface {
	ReadDir(dirname string) ([]string, error)
	ReadFile(filename string) ([]byte, error)
}
