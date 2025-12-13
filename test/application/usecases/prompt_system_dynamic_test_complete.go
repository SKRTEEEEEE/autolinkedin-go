package usecases

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	
	"github.com/linkgen-ai/backend/src/domain/entities"
)

// MockPromptSystem mocks the prompt system for testing
type MockPromptSystem struct {
	prompts map[string]*entities.Prompt
	topics  map[string]*entities.Topic
}

func (m *MockPromptSystem) LoadFromSeed(ctx context.Context, userID string) error {
	// Mock implementation
	return nil
}

func (m *MockPromptSystem) ValidatePrompt(ctx context.Context, promptID string) error {
	prompt, exists := m.prompts[promptID]
	if !exists {
		return fmt.Errorf("prompt not found: %s", promptID)
	}
	return prompt.Validate()
}

func (m *MockPromptSystem) ProcessPrompt(ctx context.Context, promptID string, topicID string) (string, error) {
	prompt, exists := m.prompts[promptID]
	if !exists {
		return "", fmt.Errorf("prompt not found: %s", promptID)
	}
	
	topic, exists := m.topics[topicID]
	if !exists {
		return "", fmt.Errorf("topic not found: %s", topicID)
	}
	
	result := prompt.PromptTemplate
	result = strings.ReplaceAll(result, "{name}", topic.Name)
	result = strings.ReplaceAll(result, "{description}", topic.Description)
	result = strings.ReplaceAll(result, "{category}", topic.Category)
	
	if len(topic.Keywords) > 0 {
		keywordsStr := strings.Join(topic.Keywords, ", ")
		result = strings.ReplaceAll(result, "{keywords}", keywordsStr)
	} else {
		result = strings.ReplaceAll(result, "{keywords}", "")
	}
	
	return result, nil
}

func (m *MockPromptSystem) InvalidateCache(ctx context.Context, promptID string) error {
	return nil
}

func (m *MockPromptSystem) ValidateEntity(prompt *entities.Prompt) error {
	return prompt.Validate()
}

func NewMockPromptSystem() *MockPromptSystem {
	return &MockPromptSystem{
		prompts: make(map[string]*entities.Prompt),
		topics:  make(map[string]*entities.Topic),
	}
}

// TestPromptSystemDynamic tests the dynamic prompt system functionality
func TestPromptSystemDynamic(t *testing.T) {
	ctx := context.Background()
	
	t.Run("should load prompts from seed configuration", func(t *testing.T) {
		// GIVEN seed configuration with prompts
		mockSystem := NewMockPromptSystem()
		userID := primitive.NewObjectID().Hex()
		
		seedPrompts := map[string]*entities.Prompt{
			"ideas1": {
				ID:             primitive.NewObjectID().Hex(),
				UserID:         userID,
				Type:           entities.PromptTypeIdeas,
				StyleName:      "",
				PromptTemplate: "Generate creative ideas about {name}",
				Active:         true,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
			"drafts1": {
				ID:             primitive.NewObjectID().Hex(),
				UserID:         userID,
				Type:           entities.PromptTypeDrafts,
				StyleName:      "professional",
				PromptTemplate: "Create professional content: {content}",
				Active:         true,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
		}
		
		// Add prompts to mock system
		for id, prompt := range seedPrompts {
			mockSystem.prompts[id] = prompt
		}
		
		// WHEN loading prompt system
		err := mockSystem.LoadFromSeed(ctx, userID)
		require.NoError(t, err)
		
		// THEN prompts should be loaded correctly
		assert.Len(t, mockSystem.prompts, 2)
		
		// Verify prompt types
		var ideasPrompt, draftsPrompt *entities.Prompt
		for _, prompt := range mockSystem.prompts {
			if prompt.Type == entities.PromptTypeIdeas {
				ideasPrompt = prompt
			} else if prompt.Type == entities.PromptTypeDrafts {
				draftsPrompt = prompt
			}
		}
		
		assert.NotNil(t, ideasPrompt)
		assert.NotNil(t, draftsPrompt)
		assert.Equal(t, "professional", draftsPrompt.StyleName)
	})
	
	t.Run("should validate prompt templates before use", func(t *testing.T) {
		// GIVEN prompts with templates
		mockSystem := NewMockPromptSystem()
		userID := primitive.NewObjectID().Hex()
		
		validPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			StyleName:      "",
			PromptTemplate: "Generate {count} ideas about {name} with keywords {keywords}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		invalidPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			StyleName:      "",
			PromptTemplate: "Too short",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		mockSystem.prompts["valid"] = validPrompt
		mockSystem.prompts["invalid"] = invalidPrompt
		
		// WHEN validating
		err1 := mockSystem.ValidatePrompt(ctx, "valid")
		err2 := mockSystem.ValidatePrompt(ctx, "invalid")
		
		// THEN templates should be validated correctly
		require.NoError(t, err1, "Valid prompt template should pass validation")
		assert.Error(t, err2, "Invalid prompt template should fail validation")
		assert.Contains(t, err2.Error(), "too short")
	})
	
	t.Run("should handle missing prompts gracefully", func(t *testing.T) {
		// GIVEN a topic referencing a missing prompt
		mockSystem := NewMockPromptSystem()
		
		// WHEN attempting to use the prompt
		processedPrompt, err := mockSystem.ProcessPrompt(ctx, "nonexistent", "topic_id")
		
		// THEN appropriate error should be returned
		assert.Error(t, err)
		assert.Empty(t, processedPrompt)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestPromptVariableSubstitution tests variable substitution in prompts
func TestPromptVariableSubstitution(t *testing.T) {
	ctx := context.Background()
	
	t.Run("should substitute variables in prompt templates", func(t *testing.T) {
		// GIVEN prompts with templates and topics
		mockSystem := NewMockPromptSystem()
		userID := primitive.NewObjectID().Hex()
		
		prompt := &entities.Prompt{
			ID:             "prompt1",
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			StyleName:      "",
			PromptTemplate: "Generate {count} ideas about {name} focusing on {keywords}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		topic := &entities.Topic{
			ID:          "topic1",
			UserID:      userID,
			Name:        "Cloud Computing",
			Description: "Cloud infrastructure and services",
			Keywords:    []string{"aws", "azure", "gcp"},
			Category:    "Technology",
			Priority:    8,
			Active:      true,
			CreatedAt:   time.Now(),
		}
		
		mockSystem.prompts[prompt.ID] = prompt
		mockSystem.topics[topic.ID] = topic
		
		// WHEN processing the prompt with topic data
		processedPrompt, err := mockSystem.ProcessPrompt(ctx, prompt.ID, topic.ID)
		require.NoError(t, err)
		
		// THEN variables should be substituted correctly
		assert.Contains(t, processedPrompt, "Cloud Computing")
		assert.Contains(t, processedPrompt, "aws, azure, gcp")
		assert.NotContains(t, processedPrompt, "{name}")
		assert.NotContains(t, processedPrompt, "{keywords}")
	})
	
	testCases := []struct {
		name           string
		template       string
		topic          *entities.Topic
		expectedResult string
		shouldPass     bool
	}{
		{
			name:     "simple variable substitution",
			template: "Generate {count} ideas about {name}",
			topic: &entities.Topic{
				ID:   "topic1",
				Name: "Marketing Digital",
			},
			expectedResult: "Generate {count} ideas about Marketing Digital",
			shouldPass:     true,
		},
		{
			name:     "related_topics array substitution",
			template: "Consider topics: {keywords} for {name}",
			topic: &entities.Topic{
				ID:       "topic2",
				Name:     "SEO Strategy",
				Keywords: []string{"Keywords", "Backlinks", "Content"},
			},
			expectedResult: "Consider topics: Keywords, Backlinks, Content for SEO Strategy",
			shouldPass:     true,
		},
		{
			name:     "empty related topics",
			template: "Topics: {keywords}",
			topic: &entities.Topic{
				ID:       "topic3",
				Name:     "Simple Topic",
				Keywords: []string{},
			},
			expectedResult: "Topics: ",
			shouldPass:     true,
		},
		{
			name:     "missing variable in template",
			template: "Generate ideas about {name}",
			topic: &entities.Topic{
				ID:   "topic4",
				Name: "", // Empty name field
			},
			expectedResult: "Generate ideas about ",
			shouldPass:     true,
		},
		{
			name:     "complex nested template",
			template: "For topic '{name}' with {count} ideas, consider {keywords} and generate unique content",
			topic: &entities.Topic{
				ID:       "topic5",
				Name:     "AI Implementation",
				Keywords: []string{"Machine Learning", "Neural Networks", "Deep Learning"},
			},
			expectedResult: "For topic 'AI Implementation' with {count} ideas, consider Machine Learning, Neural Networks, Deep Learning and generate unique content",
			shouldPass:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockSystem := NewMockPromptSystem()
			
			prompt := &entities.Prompt{
				ID:             "testPrompt",
				UserID:         "user123",
				Type:           entities.PromptTypeIdeas,
				StyleName:      "",
				PromptTemplate: tc.template,
				Active:         true,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			
			mockSystem.prompts[prompt.ID] = prompt
			mockSystem.topics[tc.topic.ID] = tc.topic
			
			// Process the prompt
			processedPrompt, err := mockSystem.ProcessPrompt(context.Background(), prompt.ID, tc.topic.ID)
			
			if tc.shouldPass {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedResult, processedPrompt)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestPromptValidation tests validation of prompts
func TestPromptValidation(t *testing.T) {
	ctx := context.Background()
	mockSystem := NewMockPromptSystem()
	
	t.Run("should validate required fields in prompts", func(t *testing.T) {
		// GIVEN prompts with missing required fields
		promptWithoutID := &entities.Prompt{
			UserID:         "user123",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Valid template",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		// Test with empty ID
		err := promptWithoutID.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ID cannot be empty")
		
		// Test with empty userID
		promptWithoutUserID := &entities.Prompt{
			ID:             "prompt123",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Valid template",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		err = promptWithoutUserID.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be empty")
		
		// Test with empty template
		promptWithoutTemplate := &entities.Prompt{
			ID:     "prompt123",
			UserID: "user123",
			Type:   entities.PromptTypeIdeas,
			Active: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		err = promptWithoutTemplate.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt template cannot be empty")
	})
	
	t.Run("should validate template syntax", func(t *testing.T) {
		// GIVEN prompts with invalid template syntax
		promptWithShortTemplate := &entities.Prompt{
			ID:             "prompt123",
			UserID:         "user123",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Too {short}", // Less than minimum length
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		// WHEN validating the prompt
		err := promptWithShortTemplate.Validate()
		
		// THEN validation should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})
	
	t.Run("should validate maximum template length", func(t *testing.T) {
		// GIVEN extremely long templates
		userID := primitive.NewObjectID().Hex()
		
		// Create very long template
		veryLongTemplate := ""
		for i := 0; i < entities.MaxPromptTemplateLength+1; i++ {
			veryLongTemplate += "x"
		}
		
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: veryLongTemplate,
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		// WHEN validating
		err := prompt.Validate()
		
		// THEN length limits should be enforced
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})
}
