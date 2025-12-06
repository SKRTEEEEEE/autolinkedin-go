package factories

import (
	"fmt"
	"time"

	"github.com/linkgen-ai/backend/domain/entities"
)

const (
	DefaultTopicPriority = 5
)

// NewTopic creates a new Topic entity with validation and defaults
func NewTopic(id, userID, name string) (*entities.Topic, error) {
	topic := &entities.Topic{
		ID:          id,
		UserID:      userID,
		Name:        name,
		Description: "",
		Keywords:    []string{},
		Category:    "",
		Priority:    DefaultTopicPriority,
		CreatedAt:   time.Now(),
	}

	if err := topic.Validate(); err != nil {
		return nil, fmt.Errorf("topic validation failed: %w", err)
	}

	return topic, nil
}

// NewTopicWithDetails creates a new Topic with full details
func NewTopicWithDetails(id, userID, name, description, category string, keywords []string, priority int) (*entities.Topic, error) {
	topic := &entities.Topic{
		ID:          id,
		UserID:      userID,
		Name:        name,
		Description: description,
		Keywords:    keywords,
		Category:    category,
		Priority:    priority,
		CreatedAt:   time.Now(),
	}

	// Normalize keywords to remove duplicates
	topic.NormalizeKeywords()

	if err := topic.Validate(); err != nil {
		return nil, fmt.Errorf("topic validation failed: %w", err)
	}

	return topic, nil
}

// NewTopicWithKeywords creates a new Topic with keywords
func NewTopicWithKeywords(id, userID, name string, keywords []string) (*entities.Topic, error) {
	topic, err := NewTopic(id, userID, name)
	if err != nil {
		return nil, err
	}

	topic.Keywords = keywords
	topic.NormalizeKeywords()

	if err := topic.Validate(); err != nil {
		return nil, fmt.Errorf("topic validation failed: %w", err)
	}

	return topic, nil
}
