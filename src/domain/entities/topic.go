package entities

import (
	"fmt"
	"strings"
	"time"
)

// Topic represents a content topic for idea generation
type Topic struct {
	ID          string
	UserID      string
	Name        string
	Description string
	Keywords    []string
	Category    string
	Priority    int
	Active      bool
	CreatedAt   time.Time
}

const (
	MinTopicNameLength = 3
	MaxTopicNameLength = 100
	MaxCategoryLength  = 50
	MaxKeywords        = 10
	MinPriority        = 1
	MaxPriority        = 10
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

	if err := t.validateKeywords(); err != nil {
		return err
	}

	if err := t.validatePriority(); err != nil {
		return err
	}

	if t.CreatedAt.IsZero() {
		return fmt.Errorf("created timestamp cannot be zero")
	}

	if t.CreatedAt.After(time.Now()) {
		return fmt.Errorf("created timestamp cannot be in the future")
	}

	return nil
}

// validateKeywords validates keyword list
func (t *Topic) validateKeywords() error {
	if len(t.Keywords) > MaxKeywords {
		return fmt.Errorf("too many keywords (maximum %d)", MaxKeywords)
	}

	seen := make(map[string]bool)
	for _, keyword := range t.Keywords {
		if keyword == "" {
			return fmt.Errorf("keywords cannot contain empty strings")
		}

		// Remove duplicates by tracking
		normalized := strings.ToLower(strings.TrimSpace(keyword))
		if seen[normalized] {
			return fmt.Errorf("duplicate keyword found: %s", keyword)
		}
		seen[normalized] = true
	}

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

	if len(t.Keywords) > 0 {
		builder.WriteString(fmt.Sprintf("Keywords: %s\n", strings.Join(t.Keywords, ", ")))
	}

	if t.Category != "" {
		builder.WriteString(fmt.Sprintf("Category: %s\n", t.Category))
	}

	return builder.String()
}

// NormalizeKeywords removes duplicates and normalizes keywords
func (t *Topic) NormalizeKeywords() {
	seen := make(map[string]bool)
	normalized := []string{}

	for _, keyword := range t.Keywords {
		key := strings.ToLower(strings.TrimSpace(keyword))
		if key != "" && !seen[key] {
			seen[key] = true
			normalized = append(normalized, keyword)
		}
	}

	t.Keywords = normalized
}
