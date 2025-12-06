package valueobjects

import (
	"fmt"
	"strings"
	"time"
)

// RefinementEntry represents a single refinement in the draft history
// This is an immutable value object
type RefinementEntry struct {
	timestamp time.Time
	prompt    string
	content   string
	version   int
}

// NewRefinementEntry creates a new refinement entry
func NewRefinementEntry(prompt, content string, version int) (*RefinementEntry, error) {
	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "" {
		return nil, fmt.Errorf("refinement prompt cannot be empty")
	}

	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return nil, fmt.Errorf("refinement content cannot be empty")
	}

	if version < 1 {
		return nil, fmt.Errorf("refinement version must be >= 1")
	}

	return &RefinementEntry{
		timestamp: time.Now(),
		prompt:    trimmedPrompt,
		content:   trimmedContent,
		version:   version,
	}, nil
}

// Timestamp returns the refinement timestamp
func (r *RefinementEntry) Timestamp() time.Time {
	return r.timestamp
}

// Prompt returns the refinement prompt
func (r *RefinementEntry) Prompt() string {
	return r.prompt
}

// Content returns the refinement content
func (r *RefinementEntry) Content() string {
	return r.content
}

// Version returns the refinement version number
func (r *RefinementEntry) Version() int {
	return r.version
}

// Equals checks if two refinement entries are equal
func (r *RefinementEntry) Equals(other *RefinementEntry) bool {
	if other == nil {
		return false
	}

	return r.version == other.version &&
		r.prompt == other.prompt &&
		r.content == other.content &&
		r.timestamp.Equal(other.timestamp)
}
