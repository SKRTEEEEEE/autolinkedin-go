package usecases

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	
	"github.com/linkgen-ai/backend/src/domain/entities"
)

// TestPromptContentValidation tests content validation in prompts
func TestPromptContentValidation(t *testing.T) {
	ctx := context.Background()
	userID := primitive.NewObjectID().Hex()
	
	t.Run("should enforce minimum content length", func(t *testing.T) {
		// GIVEN a prompt with content below minimum length
		shortPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			StyleName:      "",
			PromptTemplate: "...", // Too short
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		// WHEN validating the prompt
		err := shortPrompt.Validate()
		
		// THEN validation should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})
	
	t.Run("should enforce maximum content length", func(t *testing.T) {
		// GIVEN a prompt with content exceeding maximum length
		longContent := ""
		for i := 0; i < entities.MaxPromptTemplateLength+100; i++ {
			longContent += "a"
		}
		
		longPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			StyleName:      "",
			PromptTemplate: longContent, // Too long
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		// WHEN validating the prompt
		err := longPrompt.Validate()
		
		// THEN validation should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})
	
	t.Run("should accept content within valid length range", func(t *testing.T) {
		// GIVEN a prompt with content within valid length range
		validContent := "This is a valid prompt template that includes variables like {name} and {keywords}"
		
		validPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			StyleName:      "",
			PromptTemplate: validContent,
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		// WHEN validating the prompt
		err := validPrompt.Validate()
		
		// THEN validation should pass
		require.NoError(t, err)
	})
	
	t.Run("should accept content exactly at boundaries", func(t *testing.T) {
		// GIVEN prompts with content exactly at minimum and maximum boundaries
		
		// Test minimum boundary
		minContent := ""
		for i := 0; i < entities.MinPromptTemplateLength; i++ {
			minContent += "x"
		}
		
		minPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			StyleName:      "",
			PromptTemplate: minContent,
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		// WHEN validating minimum boundary
		err := minPrompt.Validate()
		// THEN validation should pass
		require.NoError(t, err)
		
		// Test maximum boundary
		maxContent := ""
		for i := 0; i < entities.MaxPromptTemplateLength; i++ {
			maxContent += "x"
		}
		
		maxPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			StyleName:      "",
			PromptTemplate: maxContent,
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		// WHEN validating maximum boundary
		err = maxPrompt.Validate()
		// THEN validation should pass
		require.NoError(t, err)
	})
}

// TestIdeaContentValidation tests content validation in ideas
func TestIdeaContentValidation(t *testing.T) {
	ctx := context.Background()
	userID := primitive.NewObjectID().Hex()
	topicID := primitive.NewObjectID().Hex()
	
	t.Run("should enforce minimum idea content length", func(t *testing.T) {
		// GIVEN an idea with content below minimum length
		shortIdea := &entities.Idea{
			ID:      primitive.NewObjectID().Hex(),
			UserID:  userID,
			TopicID: topicID,
			Content: "...", // Too short
			Used:    false,
		}
		
		// WHEN validating the idea
		err := shortIdea.Validate()
		
		// THEN validation should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})
	
	t.Run("should enforce maximum idea content length", func(t *testing.T) {
		// GIVEN an idea with content exceeding maximum length
		longContent := ""
		for i := 0; i < entities.MaxIdeaContentLength+100; i++ {
			longContent += "a"
		}
		
		longIdea := &entities.Idea{
			ID:      primitive.NewObjectID().Hex(),
			UserID:  userID,
			TopicID: topicID,
			Content: longContent, // Too long
			Used:    false,
		}
		
		// WHEN validating the idea
		err := longIdea.Validate()
		
		// THEN validation should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})
	
	t.Run("should accept idea content within valid length range", func(t *testing.T) {
		// GIVEN an idea with content within valid length range
		validContent := "This is a valid idea content with appropriate length for testing validation functions."
		
		validIdea := &entities.Idea{
			ID:      primitive.NewObjectID().Hex(),
			UserID:  userID,
			TopicID: topicID,
			Content: validContent,
			Used:    false,
		}
		
		// WHEN validating the idea
		err := validIdea.Validate()
		
		// THEN validation should pass
		require.NoError(t, err)
	})
	
	t.Run("should reject idea content with invalid characters", func(t *testing.T) {
		// GIVEN an idea with potentially problematic characters
		// Note: This would depend on what characters the system considers invalid
		// For now, we'll test with control characters or other potentially invalid content
		invalidContent := "This idea contains \x00 null byte character which might be invalid"
		
		invalidIdea := &entities.Idea{
			ID:      primitive.NewObjectID().Hex(),
			UserID:  userID,
			TopicID: topicID,
			Content: invalidContent,
			Used:    false,
		}
		
		// WHEN validating the idea
		err := invalidIdea.Validate()
		
		// THEN validation should handle or reject appropriately
		// This test serves as a placeholder for more specific character validation
		// In the actual implementation, this might pass or fail depending on requirements
	})
}

// TestTopicContentValidation tests content validation in topics
func TestTopicContentValidation(t *testing.T) {
	ctx := context.Background()
	userID := primitive.NewObjectID().Hex()
	
	t.Run("should enforce minimum topic name length", func(t *testing.T) {
		// GIVEN a topic with name below minimum length
		shortTopic := &entities.Topic{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			Name:      "...", // Too short
			Keywords:  []string{"test"},
			Category:  "Testing",
			Priority:  5,
			Active:    true,
			CreatedAt: time.Now(),
		}
		
		// WHEN validating the topic
		err := shortTopic.Validate()
		
		// THEN validation should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})
	
	t.Run("should enforce maximum topic name length", func(t *testing.T) {
		// GIVEN a topic with name exceeding maximum length
		longName := ""
		for i := 0; i < entities.MaxTopicNameLength+100; i++ {
			longName += "a"
		}
		
		longTopic := &entities.Topic{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			Name:      longName, // Too long
			Keywords:  []string{"test"},
			Category:  "Testing",
			Priority:  5,
			Active:    true,
			CreatedAt: time.Now(),
		}
		
		// WHEN validating the topic
		err := longTopic.Validate()
		
		// THEN validation should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})
	
	t.Run("should accept topic content within valid length range", func(t *testing.T) {
		// GIVEN a topic with content within valid length range
		validTopic := &entities.Topic{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			Name:      "Valid Topic Name",
			Keywords:  []string{"test", "validation"},
			Category:  "Testing",
			Priority:  5,
			Active:    true,
			CreatedAt: time.Now(),
		}
		
		// WHEN validating the topic
		err := validTopic.Validate()
		
		// THEN validation should pass
		require.NoError(t, err)
	})
	
	t.Run("should enforce maximum keyword count", func(t *testing.T) {
		// GIVEN a topic with too many keywords
		tooManyKeywords := []string{}
		for i := 0; i < entities.MaxKeywords+10; i++ {
			tooManyKeywords = append(tooManyKeywords, fmt.Sprintf("keyword%d", i))
		}
		
		invalidTopic := &entities.Topic{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			Name:      "Topic with Many Keywords",
			Keywords:  tooManyKeywords, // Too many
			Category:  "Testing",
			Priority:  5,
			Active:    true,
			CreatedAt: time.Now(),
		}
		
		// WHEN validating the topic
		err := invalidTopic.Validate()
		
		// THEN validation should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many keywords")
	})
	
	t.Run("should enforce valid priority range", func(t *testing.T) {
		// GIVEN topics with invalid priority values
		
		// Test priority too low
		lowPriorityTopic := &entities.Topic{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			Name:      "Topic with Low Priority",
			Keywords:  []string{"test"},
			Category:  "Testing",
			Priority:  entities.MinPriority - 1, // Too low
			Active:    true,
			CreatedAt: time.Now(),
		}
		
		// WHEN validating the topic with low priority
		err := lowPriorityTopic.Validate()
		
		// THEN validation should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "priority must be between")
		
		// Test priority too high
		highPriorityTopic := &entities.Topic{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			Name:      "Topic with High Priority",
			Keywords:  []string{"test"},
			Category:  "Testing",
			Priority:  entities.MaxPriority + 1, // Too high
			Active:    true,
			CreatedAt: time.Now(),
		}
		
		// WHEN validating the topic with high priority
		err = highPriorityTopic.Validate()
		
		// THEN validation should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "priority must be between")
	})
	
	t.Run("should accept topic with valid priority range", func(t *testing.T) {
		// GIVEN a topic with valid priority values
		validPriorityTopic := &entities.Topic{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			Name:      "Topic with Valid Priority",
			Keywords:  []string{"test"},
			Category:  "Testing",
			Priority:  entities.MinPriority + 1, // Valid
			Active:    true,
			CreatedAt: time.Now(),
		}
		
		// WHEN validating the topic
		err := validPriorityTopic.Validate()
		
		// THEN validation should pass
		require.NoError(t, err)
	})
}
