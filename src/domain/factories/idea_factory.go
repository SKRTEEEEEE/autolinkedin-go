package factories

import (
	"fmt"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// NewIdea creates a new Idea entity with validation and expiration.
// The topicName parameter is required to ensure ideas are linked to their topic context.
func NewIdea(id, userID, topicID, topicName, content string) (*entities.Idea, error) {
	idea := &entities.Idea{
		ID:           id,
		UserID:       userID,
		TopicID:      topicID,
		TopicName:    topicName,
		Content:      content,
		QualityScore: nil,
		Used:         false,
		CreatedAt:    time.Now(),
		ExpiresAt:    nil,
	}

	// Calculate default expiration
	idea.CalculateExpiration(entities.DefaultIdeaTTLDays)

	if err := idea.Validate(); err != nil {
		return nil, fmt.Errorf("idea validation failed: %w", err)
	}

	return idea, nil
}

// NewIdeaWithTTL creates a new Idea with custom TTL
func NewIdeaWithTTL(id, userID, topicID, topicName, content string, ttlDays int) (*entities.Idea, error) {
	idea, err := NewIdea(id, userID, topicID, topicName, content)
	if err != nil {
		return nil, err
	}

	// Recalculate with custom TTL
	idea.CalculateExpiration(ttlDays)

	return idea, nil
}

// NewIdeaWithQuality creates a new Idea with quality score
func NewIdeaWithQuality(id, userID, topicID, topicName, content string, qualityScore float64) (*entities.Idea, error) {
	idea, err := NewIdea(id, userID, topicID, topicName, content)
	if err != nil {
		return nil, err
	}

	idea.QualityScore = &qualityScore

	if err := idea.Validate(); err != nil {
		return nil, fmt.Errorf("idea validation failed: %w", err)
	}

	return idea, nil
}

// NewIdeaWithoutExpiration creates a new Idea without expiration
func NewIdeaWithoutExpiration(id, userID, topicID, topicName, content string) (*entities.Idea, error) {
	idea := &entities.Idea{
		ID:           id,
		UserID:       userID,
		TopicID:      topicID,
		TopicName:    topicName,
		Content:      content,
		QualityScore: nil,
		Used:         false,
		CreatedAt:    time.Now(),
		ExpiresAt:    nil,
	}

	if err := idea.Validate(); err != nil {
		return nil, fmt.Errorf("idea validation failed: %w", err)
	}

	return idea, nil
}
