package domain

import (
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewIdea_ShouldCreateIdeaWithTopicName(t *testing.T) {
	// Given
	id := "test-idea-id"
	userID := "user-123"
	topicID := "topic-456"
	topicName := "AI Content Generation"
	content := "This is a test idea about AI content generation"

	// When
	idea, err := entities.NewIdea(id, userID, topicID, topicName, content)

	// Then
	require.NoError(t, err)
	assert.Equal(t, id, idea.ID)
	assert.Equal(t, userID, idea.UserID)
	assert.Equal(t, topicID, idea.TopicID)
	assert.Equal(t, topicName, idea.TopicName) // New field should be set
	assert.Equal(t, content, idea.Content)
	assert.False(t, idea.Used)
	assert.NotNil(t, idea.CreatedAt)
}

func Test_NewIdeaWithTTL_ShouldCreateIdeaWithTopicNameAndCustomTTL(t *testing.T) {
	// Given
	id := "test-idea-id"
	userID := "user-123"
	topicID := "topic-456"
	topicName := "AI Content Generation"
	content := "This is a test idea about AI content generation"
	ttlDays := 60

	// When
	idea, err := entities.NewIdeaWithTTL(id, userID, topicID, topicName, content, ttlDays)

	// Then
	require.NoError(t, err)
	assert.Equal(t, topicName, idea.TopicName)
	require.NotNil(t, idea.ExpiresAt)
	expectedExpiry := idea.CreatedAt.Add(time.Duration(ttlDays) * 24 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, *idea.ExpiresAt, time.Second)
}

func Test_NewIdeaWithQuality_ShouldCreateIdeaWithTopicNameAndQuality(t *testing.T) {
	// Given
	id := "test-idea-id"
	userID := "user-123"
	topicID := "topic-456"
	topicName := "AI Content Generation"
	content := "This is a test idea about AI content generation"
	qualityScore := 0.85

	// When
	idea, err := entities.NewIdeaWithQuality(id, userID, topicID, topicName, content, qualityScore)

	// Then
	require.NoError(t, err)
	assert.Equal(t, topicName, idea.TopicName)
	require.NotNil(t, idea.QualityScore)
	assert.Equal(t, qualityScore, *idea.QualityScore)
}

func Test_Idea_Validate_ShouldEnforceTopicNameConstraints(t *testing.T) {
	// Given
	idea := &entities.Idea{
		ID:        "test-idea-id",
		UserID:    "user-123",
		TopicID:   "topic-456",
		Content:   "Valid content that meets minimum length requirements",
		TopicName: "", // Invalid: empty topic name
		CreatedAt: time.Now(),
	}

	// When
	err := idea.Validate()

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic name cannot be empty")
}

func Test_Idea_Validate_ShouldEnforceTopicNameLength(t *testing.T) {
	// Given
	longTopicName := "This is a very long topic name that exceeds the maximum allowed length for topic names in the system and should trigger validation error"

	idea := &entities.Idea{
		ID:        "test-idea-id",
		UserID:    "user-123",
		TopicID:   "topic-456",
		Content:   "Valid content that meets minimum length requirements",
		TopicName: longTopicName,
		CreatedAt: time.Now(),
	}

	// When
	err := idea.Validate()

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic name too long")
}

func Test_Idea_UpdateContent_ShouldPreserveTopicName(t *testing.T) {
	// Given
	idea, _ := entities.NewIdea("test-id", "user-123", "topic-456", "Original Topic", "Original content")
	newContent := "Updated content with more details"

	// When
	err := idea.UpdateContent(newContent)

	// Then
	require.NoError(t, err)
	assert.Equal(t, newContent, idea.Content)
	assert.Equal(t, "Original Topic", idea.TopicName) // Should remain unchanged
}
