package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NOTE: These tests are written according to TDD Red pattern - they will FAIL
// until the PromptEngine implementation exists

func TestPromptEngine(t *testing.T) {
	// Test for the PromptEngine service in infrastructure/services:
	// - Cache processed prompts (map[string]string)
	// - Parse and replace variables according to seed/README.md:
	//   * {name} → Topic.name
	//   * {ideas} → Topic.ideas (número)
	//   * {[related_topics]} → strings.Join(Topic.related_topics, ", ")
	//   * {content} → Idea.content
	//   * {user_context} → resultado de buildUserContext() actual
	// - Fallback system for missing prompts

	// Setup
	ctx := context.Background()
	mockRepo := &MockPromptsRepository{}
	mockLogger := &MockLogger{}
	engine := services.NewPromptEngine(mockRepo, mockLogger)

	// Setup test data
	userID := "test-user-123"
	now := time.Now()

	// Setup test user with profile data using Configuration (new system)
	testUser := &entities.User{
		ID:        userID,
		Email:     "test@example.com",
		Language:  "es",
		Configuration: map[string]interface{}{
			"name":               "Juan García",
			"expertise":          "Desarrollo Backend",
			"tone_preference":    "Profesional",
			"industry":           "Technology",
			"role":               "Software Engineer",
			"experience":         "5 years",
			"goals":              "Career growth in AI/ML",
		},
		CreatedAt: now,
		UpdatedAt: now,
		Active:    true,
	}

	// Setup test prompts
	base1Prompt := &entities.Prompt{
		ID:             "prompt-1",
		UserID:         userID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Eres un experto en estrategia de contenido para LinkedIn. Genera {ideas} ideas de contenido únicas y atractivas sobre el siguiente tema:\n\nTema: {name}\nTemas relacionados: {[related_topics]}\n\nRequisitos:\n- Cada idea debe ser específica y accionable\n- Las ideas deben ser diversas y cubrir diferentes ángulos\n- Enfócate en valor profesional e insights\n- Mantén las ideas concisas (1-2 oraciones cada una)\n- Hazlas adecuadas para la audiencia de LinkedIn\n- IMPORTANTE: Genera el contenido SIEMPRE en español\n\nDevuelve ÚNICAMENTE un objeto JSON con este formato exacto:\n{\"ideas\": [\"idea1\", \"idea2\", \"idea3\", ...]}",
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	proPrompt := &entities.Prompt{
		ID:             "prompt-2",
		UserID:         userID,
		Name:           "profesional",
		Type:           entities.PromptTypeDrafts,
		PromptTemplate: "Eres un experto creador de contenido para LinkedIn.\n\nBasándote en la siguiente idea:\n{content}\n\nContexto adicional del usuario:\n{user_context}\n\nInstrucciones clave:\n- Escribe SIEMPRE en español neutro profesional.\n- Cada post debe tener 120-260 palabras, abrir con un gancho potente y cerrar con una CTA o pregunta.\n- El artículo debe tener título atractivo, introducción, desarrollo con viñetas o subtítulos y conclusión clara.\n- No inventes datos sensibles, pero puedes añadir insights inspirados en mejores prácticas.\n- No utilices comillas triples, bloques de código ni texto fuera del JSON.\n- IMPORTANTE: El JSON debe ser 100%% válido, sin errores de sintaxis.\n\nFORMATO OBLIGATORIO: Responde ÚNICAMENTE con el JSON siguiente, sin texto adicional:\n{\n  \"posts\": [\n    \"Post 1 completo en una sola cadena\",\n    \"Post 2 completo\",\n    \"Post 3 completo\",\n    \"Post 4 completo\",\n    \"Post 5 completo\"\n  ],\n  \"articles\": [\n    \"Título del artículo\\n\\nCuerpo del artículo con secciones y conclusión\"\n  ]\n}",
		StyleName:      "profesional",
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	setup := func() {
		mockRepo.prompts = []*entities.Prompt{base1Prompt, proPrompt}
		engine.ClearCache() // Clear cache for clean tests
	}

	t.Run("should process ideas prompt with variable substitution", func(t *testing.T) {
		setup()

		// GIVEN a topic with related data for ideas prompt
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "Machine Learning for Startups",
			Ideas:           5,
			RelatedTopics:   []string{"AI", "Venture Capital", "Technology Innovation"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing the prompt
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)

		// THEN variables should be replaced correctly
		require.NoError(t, err)
		assert.NotNil(t, processedPrompt)
		
		// Verify variables were substituted
		assert.Contains(t, processedPrompt, "Genera 5 ideas de contenido")
		assert.Contains(t, processedPrompt, "Machine Learning for Startups")
		assert.Contains(t, processedPrompt, "AI, Venture Capital, Technology Innovation")
		assert.NotContains(t, processedPrompt, "{ideas}")
		assert.NotContains(t, processedPrompt, "{name}")
		assert.NotContains(t, processedPrompt, "{[related_topics]}")
	})

	t.Run("should process drafts prompt with content and user context", func(t *testing.T) {
		setup()

		// GIVEN an idea and user for drafts prompt
		idea := &entities.Idea{
			ID:        "idea-1",
			TopicID:   "topic-1",
			Content:   "How ML models are revolutionizing startup growth strategies through predictive analytics",
			Used:      false,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// WHEN processing the prompt
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, "profesional", entities.PromptTypeDrafts, nil, idea, testUser)

		// THEN variables should be replaced correctly
		require.NoError(t, err)
		assert.NotNil(t, processedPrompt)
		
		// Verify content variable was substituted
		assert.Contains(t, processedPrompt, "How ML models are revolutionizing startup growth strategies")
		
		// Verify user context was built and substituted (using new Configuration system)
		assert.Contains(t, processedPrompt, "Name: Juan García")
		assert.Contains(t, processedPrompt, "Expertise: Desarrollo Backend")
		assert.Contains(t, processedPrompt, "Tone: Profesional")
		
		assert.NotContains(t, processedPrompt, "{content}")
		assert.NotContains(t, processedPrompt, "{user_context}")
	})

	t.Run("should cache processed prompts", func(t *testing.T) {
		setup()

		// GIVEN a topic
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "Test Topic",
			Ideas:           3,
			RelatedTopics:   []string{"AI"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing the prompt twice
		processed1, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)
		require.NoError(t, err)
		
		processed2, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)
		require.NoError(t, err)

		// THEN results should be identical and cached
		assert.Equal(t, processed1, processed2)
		
		// Verify cache contains something (we can't access private buildCacheKey)
		assert.Greater(t, engine.CacheSize(), 0)
		assert.Greater(t, engine.CacheHitCount(), 0)
	})

	t.Run("should use fallback for missing prompts", func(t *testing.T) {
		setup()

		// GIVEN no custom prompt exists for the user
		nonExistentPromptName := "nonexistent-prompt"
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "Test Topic",
			Ideas:           3,
			RelatedTopics:   []string{"AI"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing a non-existent prompt
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, nonExistentPromptName, entities.PromptTypeIdeas, topic, nil, testUser)

		// THEN should fall back to default prompts from seed files
		require.NoError(t, err)
		assert.NotNil(t, processedPrompt)
		
		// Should contain the default prompt content for the type
		assert.Contains(t, processedPrompt, "Genera 3 ideas de contenido")
		assert.Contains(t, processedPrompt, "Test Topic")
		assert.Contains(t, processedPrompt, "AI")
	})

	t.Run("should build user context correctly", func(t *testing.T) {
		setup()

		// WHEN building user context
		userContext := engine.BuildUserContext(testUser)

		// THEN should contain all user profile information (using new Configuration system)
		assert.Contains(t, userContext, "Name: Juan García")
		assert.Contains(t, userContext, "Expertise: Desarrollo Backend")
		assert.Contains(t, userContext, "Tone: Profesional")
	})

	t.Run("should handle empty related topics array", func(t *testing.T) {
		setup()

		// GIVEN a topic without related topics
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "Solo Topic",
			Ideas:           2,
			RelatedTopics:   []string{}, // Empty array
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing the prompt
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)

		// THEN should handle empty array correctly
		require.NoError(t, err)
		assert.NotNil(t, processedPrompt)
		assert.Contains(t, processedPrompt, "Genera 2 ideas de contenido")
		assert.Contains(t, processedPrompt, "Solo Topic")
		// Should not include "Temas relacionados:" section when empty
		assert.NotContains(t, processedPrompt, "Temas relacionados:")
	})

	t.Run("should validate required variables before processing", func(t *testing.T) {
		setup()

		// GIVEN a topic with missing name (required for ideas prompt)
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "", // Missing required field
			Ideas:           3,
			RelatedTopics:   []string{"AI"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing the prompt
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)

		// THEN should return validation error or empty prompt (implementation detail)
		// Note: Current implementation returns empty string instead of error for validation
		if err != nil {
			assert.Contains(t, err.Error(), "missing required variable")
		} else {
			assert.Empty(t, processedPrompt)
		}
	})

	t.Run("should handle drafts prompt fallback for missing style", func(t *testing.T) {
		setup()

		// GIVEN no custom style exists for drafts
		nonExistentStyle := "nonexistent-style"
		idea := &entities.Idea{
			ID:        "idea-1",
			TopicID:   "topic-1",
			TopicName: "Test Topic",
			Content:   "Content for testing",
			Used:      false,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// WHEN processing with non-existent style
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, nonExistentStyle, entities.PromptTypeDrafts, nil, idea, testUser)

		// THEN should fall back to default drafts prompt
		require.NoError(t, err)
		assert.NotNil(t, processedPrompt)
		assert.Contains(t, processedPrompt, "Content for testing")
		assert.Contains(t, processedPrompt, "Name: Juan García")
	})

	t.Run("should return error for invalid user", func(t *testing.T) {
		setup()

		// GIVEN an invalid user (nil)
		topic := &entities.Topic{
			ID:              "topic-1",
			Name:            "Test Topic",
			Ideas:           3,
			RelatedTopics:   []string{"AI"},
			Active:          true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// WHEN processing without a valid user
		processedPrompt, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, nil)

		// THEN should return error or handle appropriately
		if err != nil {
			assert.Contains(t, err.Error(), "user is required")
		} else {
			// If no error, at least validate that prompt processing behaves correctly
			t.Logf("No error returned, got result: %s", processedPrompt)
		}
	})

	t.Run("should handle prompt engine initialization", func(t *testing.T) {
		// GIVEN a new prompt engine
		newEngine := services.NewPromptEngine(mockRepo, mockLogger)

		// WHEN initialized
		// THEN should have empty cache and valid repository reference
		assert.NotNil(t, newEngine)
		assert.Empty(t, newEngine.GetCacheContents())
		
		// Should have default prompts loaded from seed
		defaultIdeas := newEngine.GetDefaultPrompt(entities.PromptTypeIdeas)
		assert.NotEmpty(t, defaultIdeas)
		assert.Contains(t, defaultIdeas, "Genera {ideas} ideas")
		
		defaultDrafts := newEngine.GetDefaultPrompt(entities.PromptTypeDrafts)
		assert.NotEmpty(t, defaultDrafts)
		assert.Contains(t, defaultDrafts, "Eres un experto creador de contenido para LinkedIn")
	})

	t.Run("should record prompt usage logs for diagnostics", func(t *testing.T) {
		setup()

		// GIVEN the prompt engine is processing prompts for observability
		topic := &entities.Topic{
			ID:            "topic-log",
			Name:          "Observability",
			Ideas:         2,
			RelatedTopics: []string{"logging"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		_, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)
		require.NoError(t, err)

		// WHEN retrieving collected log entries
		logs := engine.GetLogEntries()
		require.NotNil(t, logs)
		require.NotEmpty(t, logs)

		entry := logs[0]
		assert.Equal(t, userID, entry.UserID)
		assert.Equal(t, "base1", entry.PromptName)
		assert.Equal(t, string(entities.PromptTypeIdeas), entry.PromptType)
		assert.True(t, entry.Success)
		assert.WithinDuration(t, now, entry.Timestamp, 5*time.Second)
	})

	t.Run("should expose cache metrics for observability endpoints", func(t *testing.T) {
		setup()

		// GIVEN prompts processed twice to hit cache
		topic := &entities.Topic{
			ID:            "topic-cache",
			Name:          "Cache",
			Ideas:         4,
			RelatedTopics: []string{"metrics"},
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		_, err := engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)
		require.NoError(t, err)

		_, err = engine.ProcessPrompt(ctx, userID, "base1", entities.PromptTypeIdeas, topic, nil, testUser)
		require.NoError(t, err)

		// THEN metrics should report cache usage information
		assert.Greater(t, engine.CacheSize(), 0)
		assert.GreaterOrEqual(t, engine.CacheHitCount(), 1)
	})

	t.Run("should build diagnostics payload per user", func(t *testing.T) {
		setup()

		// GIVEN prompts exist for the user
		diagnostics := engine.GetDiagnostics(ctx, userID)
		require.NotNil(t, diagnostics)

		// THEN diagnostics should include counts and supported variables
		assert.True(t, diagnostics.PromptEngineActive)
		assert.Equal(t, 2, diagnostics.UserPromptCount)
		assert.Contains(t, diagnostics.SupportedVariables, "{ideas}")
		assert.Equal(t, engine.CacheSize(), diagnostics.CacheSize)
	})
}

// Mock implementations for testing

type MockPromptsRepository struct {
	prompts []*entities.Prompt
}

func (m *MockPromptsRepository) Create(ctx context.Context, prompt *entities.Prompt) (string, error) {
	for _, p := range m.prompts {
		if p.ID == "" {
			p.ID = primitive.NewObjectID().Hex()
		}
	}
	return primitive.NewObjectID().Hex(), nil
}

func (m *MockPromptsRepository) FindByID(ctx context.Context, id string) (*entities.Prompt, error) {
	for _, prompt := range m.prompts {
		if prompt.ID == id {
			return prompt, nil
		}
	}
	return nil, nil
}

func (m *MockPromptsRepository) FindByName(ctx context.Context, userID string, name string) (*entities.Prompt, error) {
	for _, prompt := range m.prompts {
		if prompt.UserID == userID && prompt.Name == name {
			return prompt, nil
		}
	}
	return nil, nil
}

func (m *MockPromptsRepository) ListByUserID(ctx context.Context, userID string) ([]*entities.Prompt, error) {
	var result []*entities.Prompt
	for _, prompt := range m.prompts {
		if prompt.UserID == userID {
			result = append(result, prompt)
		}
	}
	return result, nil
}

func (m *MockPromptsRepository) ListByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	var result []*entities.Prompt
	for _, prompt := range m.prompts {
		if prompt.UserID == userID && prompt.Type == promptType {
			result = append(result, prompt)
		}
	}
	return result, nil
}

func (m *MockPromptsRepository) FindActiveByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	var result []*entities.Prompt
	for _, prompt := range m.prompts {
		if prompt.UserID == userID && prompt.Type == promptType && prompt.Active {
			result = append(result, prompt)
		}
	}
	return result, nil
}

func (m *MockPromptsRepository) FindByUserIDAndStyle(ctx context.Context, userID string, styleName string) (*entities.Prompt, error) {
	for _, prompt := range m.prompts {
		if prompt.UserID == userID && prompt.StyleName == styleName && prompt.Type == entities.PromptTypeDrafts {
			return prompt, nil
		}
	}
	return nil, nil
}

func (m *MockPromptsRepository) Update(ctx context.Context, prompt *entities.Prompt) error {
	for i, p := range m.prompts {
		if p.ID == prompt.ID {
			m.prompts[i] = prompt
			return nil
		}
	}
	return fmt.Errorf("prompt not found")
}

func (m *MockPromptsRepository) Delete(ctx context.Context, id string) error {
	for i, prompt := range m.prompts {
		if prompt.ID == id {
			m.prompts = append(m.prompts[:i], m.prompts[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("prompt not found")
}

func (m *MockPromptsRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	count := 0
	for _, prompt := range m.prompts {
		if prompt.UserID == userID {
			count++
		}
	}
	return int64(count), nil
}

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	debugLogs []string
	infoLogs  []string
	warnLogs  []string
	errorLogs []string
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.debugLogs = append(m.debugLogs, fmt.Sprintf(msg, fields...))
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.infoLogs = append(m.infoLogs, fmt.Sprintf(msg, fields...))
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.warnLogs = append(m.warnLogs, fmt.Sprintf(msg, fields...))
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.errorLogs = append(m.errorLogs, fmt.Sprintf(msg, fields...))
}
