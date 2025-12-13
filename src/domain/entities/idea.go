package entities

import (
	"fmt"
	"strings"
	"time"
)

// Idea represents a generated content idea
type Idea struct {
	ID           string
	UserID       string
	TopicID      string
	TopicName    string // Name of the related topic
	Content      string
	QualityScore *float64
	Used         bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    *time.Time
}

const (
	MinIdeaContentLength = 10
	MaxIdeaContentLength = 200 // Updated from 5000 to 200 as specified in entity.md
	DefaultIdeaTTLDays   = 30
)

// Validate validates the idea entity
func (i *Idea) Validate() error {
	if i.ID == "" {
		return fmt.Errorf("idea ID cannot be empty")
	}

	if i.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if i.TopicID == "" {
		return fmt.Errorf("topic ID cannot be empty")
	}

	if i.TopicName == "" {
		return fmt.Errorf("topic name cannot be empty")
	}

	if err := i.ValidateContent(); err != nil {
		return err
	}

	if err := i.validateQualityScore(); err != nil {
		return err
	}

	if i.CreatedAt.IsZero() {
		return fmt.Errorf("created timestamp cannot be zero")
	}

	if i.CreatedAt.After(time.Now()) {
		return fmt.Errorf("created timestamp cannot be in the future")
	}

	if i.UpdatedAt.IsZero() {
		return fmt.Errorf("updated timestamp cannot be zero")
	}

	if i.UpdatedAt.Before(i.CreatedAt) {
		return fmt.Errorf("updated timestamp cannot be before created timestamp")
	}

	if i.UpdatedAt.After(time.Now()) {
		return fmt.Errorf("updated timestamp cannot be in the future")
	}

	return nil
}

// ValidateContent validates idea content
func (i *Idea) ValidateContent() error {
	if i.Content == "" {
		return fmt.Errorf("idea content cannot be empty")
	}

	trimmed := strings.TrimSpace(i.Content)
	if trimmed == "" {
		return fmt.Errorf("idea content cannot be only whitespace")
	}

	if len(trimmed) < MinIdeaContentLength {
		return fmt.Errorf("idea content too short (minimum %d characters)", MinIdeaContentLength)
	}

	if len(i.Content) > MaxIdeaContentLength {
		return fmt.Errorf("idea content too long (maximum %d characters)", MaxIdeaContentLength)
	}

	return nil
}

// validateQualityScore validates quality score range
func (i *Idea) validateQualityScore() error {
	if i.QualityScore == nil {
		defaultScore := 0.0
		i.QualityScore = &defaultScore
	}

	if i.QualityScore != nil {
		score := *i.QualityScore
		if score < 0.0 || score > 1.0 {
			return fmt.Errorf("quality score must be between 0.0 and 1.0")
		}
	}
	return nil
}

// MarkAsUsed marks the idea as used in a draft
func (i *Idea) MarkAsUsed() error {
	if i.Used {
		return fmt.Errorf("idea is already marked as used")
	}

	i.Used = true
	return nil
}

// IsExpired checks if idea has expired
func (i *Idea) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}

	return time.Now().After(*i.ExpiresAt) || time.Now().Equal(*i.ExpiresAt)
}

// BelongsToUser checks if idea belongs to specified user
func (i *Idea) BelongsToUser(userID string) bool {
	return i.UserID != "" && i.UserID == userID
}

// CanBeUsed checks if idea can be used for draft creation
func (i *Idea) CanBeUsed() bool {
	return !i.Used && !i.IsExpired()
}

// CalculateExpiration sets expiration based on TTL in days
func (i *Idea) CalculateExpiration(ttlDays int) {
	if ttlDays <= 0 {
		ttlDays = DefaultIdeaTTLDays
	}

	expiresAt := i.CreatedAt.Add(time.Duration(ttlDays) * 24 * time.Hour)
	i.ExpiresAt = &expiresAt
}

// SetTopicName sets the topic name for the idea
func (i *Idea) SetTopicName(topicName string) {
	i.TopicName = topicName
}
