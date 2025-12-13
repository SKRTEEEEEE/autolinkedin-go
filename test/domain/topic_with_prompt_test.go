package domain

import (
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewTopic_ShouldCreateTopicWithDefaultValues(t *testing.T) {
	// Given
	id := "test-topic-id"
	userID := "user-123"
	name := "Test Topic"

	// When
	topic, err := entities.NewTopic(id, userID, name)

	// Then
	require.NoError(t, err)
	assert.Equal(t, id, topic.ID)
	assert.Equal(t, userID, topic.UserID)
	assert.Equal(t, name, topic.Name)
	assert.Equal(t, 2, topic.Ideas) // Default value should be 2
	assert.Equal(t, "base1", topic.Prompt) // Default prompt should be base1
	assert.True(t, topic.Active) // Default should be active
}

func Test_NewTopicWithDetails_ShouldCreateTopicWithCustomValues(t *testing.T) {
	// Given
	id := "test-topic-id"
	userID := "user-123"
	name := "Test Topic"
	description := "A test description"
	category := "Technology"
	keywords := []string{"AI", "LLM", "Content"}
	priority := 8
	ideas := 5
	prompt := "custom-prompt"

	// When
	topic, err := entities.NewTopicWithDetails(id, userID, name, description, category, keywords, priority, ideas, prompt)

	// Then
	require.NoError(t, err)
	assert.Equal(t, id, topic.ID)
	assert.Equal(t, userID, topic.UserID)
	assert.Equal(t, name, topic.Name)
	assert.Equal(t, description, topic.Description)
	assert.Equal(t, category, topic.Category)
	assert.Equal(t, keywords, topic.Keywords)
	assert.Equal(t, priority, topic.Priority)
	assert.Equal(t, ideas, topic.Ideas)
	assert.Equal(t, prompt, topic.Prompt)
	assert.True(t, topic.Active)
}

func Test_Topic_Validate_ShouldEnforceIdeasFieldConstraints(t *testing.T) {
	// Given
	id := "test-topic-id"
	userID := "user-123"
	name := "Test Topic"

	topic := &entities.Topic{
		ID:     id,
		UserID: userID,
		Name:   name,
		Ideas:  -1, // Invalid: negative value
	}

	// When
	err := topic.Validate()

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ideas must be between")
}

func Test_Topic_Validate_ShouldEnforcePromptFieldConstraints(t *testing.T) {
	// Given
	id := "test-topic-id"
	userID := "user-123"
	name := "Test Topic"

	topic := &entities.Topic{
		ID:     id,
		UserID: userID,
		Name:   name,
		Ideas:  2,
		Prompt: "", // Invalid: empty prompt
	}

	// When
	err := topic.Validate()

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt cannot be empty")
}

func Test_Topic_SetIdeas_ShouldUpdateIdeasCount(t *testing.T) {
	// Given
	topic, _ := entities.NewTopic("test-id", "user-123", "Test Topic")

	// When
	topic.SetIdeas(5)

	// Then
	assert.Equal(t, 5, topic.Ideas)
}

func Test_Topic_SetPrompt_ShouldUpdatePromptReference(t *testing.T) {
	// Given
	topic, _ := entities.NewTopic("test-id", "user-123", "Test Topic")

	// When
	topic.SetPrompt("custom-prompt")

	// Then
	assert.Equal(t, "custom-prompt", topic.Prompt)
}
