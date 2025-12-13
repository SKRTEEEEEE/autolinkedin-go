package entities

import (
	"fmt"
	"strings"
	"time"
)

// Topic represents a content topic for idea generation
type Topic struct {
	ID            string
	UserID        string
	Name          string
	Description   string
	Category      string // Default: "General"
	Priority      int    // 1-10, Default: 5
	Ideas         int    // Number of ideas to generate, Default: 2
	Prompt        string // Reference to prompt.name, Default: "base1"
	RelatedTopics []string
	Active        bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

const (
	MinTopicNameLength = 3
	MaxTopicNameLength = 100
	MaxCategoryLength  = 50
	MaxRelatedTopics   = 10
	MinPriority        = 1
	MaxPriority        = 10
	MinIdeasCount      = 1
	MaxIdeasCount      = 20
	DefaultCategory    = "General"
	DefaultPriority    = 5
	DefaultIdeasCount  = 2
	DefaultPrompt      = "base1"
)

// Validate ensures topic data integrity
func (t *Topic) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("topic ID cannot be empty")
	}

	if t.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if t.Name == "" {
		return fmt.Errorf("topic name cannot be empty")
	}

	if len(t.Name) < MinTopicNameLength {
		return fmt.Errorf("topic name too short (minimum %d characters)", MinTopicNameLength)
	}

	if len(t.Name) > MaxTopicNameLength {
		return fmt.Errorf("topic name too long (maximum %d characters)", MaxTopicNameLength)
	}

	if t.Category != "" && len(t.Category) > MaxCategoryLength {
		return fmt.Errorf("category too long (maximum %d characters)", MaxCategoryLength)
	}

	if err := t.validateRelatedTopics(); err != nil {
		return err
	}

	if err := t.validatePriority(); err != nil {
		return err
	}

	if err := t.validateIdeasCount(); err != nil {
		return err
	}

	if err := t.validatePrompt(); err != nil {
		return err
	}

	if t.CreatedAt.IsZero() {
		return fmt.Errorf("created timestamp cannot be zero")
	}

	if t.CreatedAt.After(time.Now()) {
		return fmt.Errorf("created timestamp cannot be in the future")
	}

	if t.UpdatedAt.IsZero() {
		return fmt.Errorf("updated timestamp cannot be zero")
	}

	if t.UpdatedAt.Before(t.CreatedAt) {
		return fmt.Errorf("updated timestamp cannot be before created timestamp")
	}

	if t.UpdatedAt.After(time.Now()) {
		return fmt.Errorf("updated timestamp cannot be in the future")
	}

	return nil
}

// validateRelatedTopics validates related topics list
func (t *Topic) validateRelatedTopics() error {
	if len(t.RelatedTopics) > MaxRelatedTopics {
		return fmt.Errorf("too many related topics (maximum %d)", MaxRelatedTopics)
	}

	seen := make(map[string]bool)
	for _, topic := range t.RelatedTopics {
		if topic == "" {
			return fmt.Errorf("related topics cannot contain empty strings")
		}

		// Remove duplicates by tracking
		normalized := strings.ToLower(strings.TrimSpace(topic))
		if seen[normalized] {
			return fmt.Errorf("duplicate related topic found: %s", topic)
		}
		seen[normalized] = true
	}

	return nil
}

// validateIdeasCount validates the ideas count field
func (t *Topic) validateIdeasCount() error {
	if t.Ideas == 0 {
		t.Ideas = DefaultIdeasCount
		return nil
	}

	if t.Ideas < MinIdeasCount || t.Ideas > MaxIdeasCount {
		return fmt.Errorf("ideas count must be between %d and %d", MinIdeasCount, MaxIdeasCount)
	}

	return nil
}

// validatePrompt validates the prompt reference field
func (t *Topic) validatePrompt() error {
	if t.Prompt == "" {
		t.Prompt = DefaultPrompt
		return nil
	}

	trimmed := strings.TrimSpace(t.Prompt)
	if trimmed == "" {
		return fmt.Errorf("prompt cannot be only whitespace")
	}

	t.Prompt = trimmed
	return nil
}

// validatePriority validates priority value
func (t *Topic) validatePriority() error {
	if t.Priority < MinPriority || t.Priority > MaxPriority {
		return fmt.Errorf("priority must be between %d and %d", MinPriority, MaxPriority)
	}

	return nil
}

// IsOwnedBy checks if topic belongs to specified user
func (t *Topic) IsOwnedBy(userID string) bool {
	return t.UserID != "" && t.UserID == userID
}

// GeneratePromptContext creates LLM context from topic
func (t *Topic) GeneratePromptContext() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Topic: %s\n", t.Name))

	if t.Description != "" {
		builder.WriteString(fmt.Sprintf("Description: %s\n", t.Description))
	}

	if len(t.RelatedTopics) > 0 {
		builder.WriteString(fmt.Sprintf("Related Topics: %s\n", strings.Join(t.RelatedTopics, ", ")))
	}

	if t.Category != "" {
		builder.WriteString(fmt.Sprintf("Category: %s\n", t.Category))
	}

	if t.Priority != DefaultPriority {
		builder.WriteString(fmt.Sprintf("Priority: %d\n", t.Priority))
	}

	if t.Ideas != DefaultIdeasCount {
		builder.WriteString(fmt.Sprintf("Ideas Count: %d\n", t.Ideas))
	}

	if t.Prompt != DefaultPrompt {
		builder.WriteString(fmt.Sprintf("Prompt: %s\n", t.Prompt))
	}

	return builder.String()
}

// NormalizeRelatedTopics removes duplicates and normalizes related topics
func (t *Topic) NormalizeRelatedTopics() {
	seen := make(map[string]bool)
	normalized := []string{}

	for _, topic := range t.RelatedTopics {
		key := strings.ToLower(strings.TrimSpace(topic))
		if key != "" && !seen[key] {
			seen[key] = true
			normalized = append(normalized, topic)
		}
	}

	t.RelatedTopics = normalized
}

// SetDefaults sets default values for empty fields
func (t *Topic) SetDefaults() {
	if t.Category == "" {
		t.Category = DefaultCategory
	}
	if t.Priority == 0 {
		t.Priority = DefaultPriority
	}
	if t.Ideas == 0 {
		t.Ideas = DefaultIdeasCount
	}
	if t.Prompt == "" {
		t.Prompt = DefaultPrompt
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = t.CreatedAt
	}
}
