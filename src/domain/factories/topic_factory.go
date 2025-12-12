package factories

import (
	"fmt"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// NewTopic creates a new Topic entity with validation and defaults
func NewTopic(id, userID, name string) (*entities.Topic, error) {
	topic := &entities.Topic{
		ID:            id,
		UserID:        userID,
		Name:          name,
		Description:   "",
		Category:      "",
		Priority:      entities.DefaultPriority,
		Ideas:         entities.DefaultIdeasCount,
		Prompt:        entities.DefaultPrompt,
		RelatedTopics: []string{},
		Active:        true,
		CreatedAt:     time.Now(),
	}

	// Set defaults and validate
	topic.SetDefaults()
	if err := topic.Validate(); err != nil {
		return nil, fmt.Errorf("topic validation failed: %w", err)
	}

	return topic, nil
}

// NewTopicWithDetails creates a new Topic with full details
func NewTopicWithDetails(id, userID, name, description, category string, priority, ideas int, prompt string, relatedTopics []string) (*entities.Topic, error) {
	topic := &entities.Topic{
		ID:            id,
		UserID:        userID,
		Name:          name,
		Description:   description,
		Category:      category,
		Priority:      priority,
		Ideas:         ideas,
		Prompt:        prompt,
		RelatedTopics: relatedTopics,
		Active:        true,
		CreatedAt:     time.Now(),
	}

	// Normalize related topics to remove duplicates
	topic.NormalizeRelatedTopics()

	// Set defaults for empty fields and validate
	topic.SetDefaults()
	if err := topic.Validate(); err != nil {
		return nil, fmt.Errorf("topic validation failed: %w", err)
	}

	return topic, nil
}

// NewTopicWithRelatedTopics creates a new Topic with related topics
func NewTopicWithRelatedTopics(id, userID, name string, relatedTopics []string) (*entities.Topic, error) {
	topic, err := NewTopic(id, userID, name)
	if err != nil {
		return nil, err
	}

	topic.RelatedTopics = relatedTopics
	topic.NormalizeRelatedTopics()

	if err := topic.Validate(); err != nil {
		return nil, fmt.Errorf("topic validation failed: %w", err)
	}

	return topic, nil
}

// NewTopicWithPrompt creates a new Topic with specific prompt reference
func NewTopicWithPrompt(id, userID, name, prompt string) (*entities.Topic, error) {
	topic, err := NewTopic(id, userID, name)
	if err != nil {
		return nil, err
	}

	topic.Prompt = prompt

	if err := topic.Validate(); err != nil {
		return nil, fmt.Errorf("topic validation failed: %w", err)
	}

	return topic, nil
}
