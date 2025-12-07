package handlers

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// ObjectID regex pattern (MongoDB ObjectId: 24 hex characters)
	objectIDRegex = regexp.MustCompile(`^[a-fA-F0-9]{24}$`)
)

// GenerateDraftRequest represents the request for draft generation
type GenerateDraftRequest struct {
	UserID string `json:"user_id"`
	IdeaID string `json:"idea_id,omitempty"`
}

// Validate validates the GenerateDraftRequest
func (r *GenerateDraftRequest) Validate() error {
	r.UserID = strings.TrimSpace(r.UserID)
	r.IdeaID = strings.TrimSpace(r.IdeaID)

	if r.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	if !isValidObjectID(r.UserID) {
		return fmt.Errorf("invalid user_id format")
	}

	// IdeaID is optional, but if provided, must be valid
	if r.IdeaID != "" && !isValidObjectID(r.IdeaID) {
		return fmt.Errorf("invalid idea_id format")
	}

	return nil
}

// RefineDraftRequest represents the request for draft refinement
type RefineDraftRequest struct {
	Prompt string `json:"prompt"`
}

// Validate validates the RefineDraftRequest
func (r *RefineDraftRequest) Validate() error {
	r.Prompt = strings.TrimSpace(r.Prompt)

	if r.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if len(r.Prompt) < 10 {
		return fmt.Errorf("prompt must be at least 10 characters")
	}

	if len(r.Prompt) > 500 {
		return fmt.Errorf("prompt exceeds maximum of 500 characters")
	}

	return nil
}

// ListIdeasRequest represents query parameters for listing ideas
type ListIdeasRequest struct {
	Topic string
	Limit int
}

// Validate validates the ListIdeasRequest
func (r *ListIdeasRequest) Validate() error {
	if r.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}

	if r.Limit > 1000 {
		return fmt.Errorf("limit exceeds maximum of 1000")
	}

	return nil
}

// ListDraftsRequest represents query parameters for listing drafts
type ListDraftsRequest struct {
	Status string
	Type   string
	Limit  int
}

// Validate validates the ListDraftsRequest
func (r *ListDraftsRequest) Validate() error {
	// Status is optional, but if provided must be valid
	if r.Status != "" {
		validStatuses := map[string]bool{
			"DRAFT":     true,
			"REFINED":   true,
			"PUBLISHED": true,
			"FAILED":    true,
		}
		if !validStatuses[strings.ToUpper(r.Status)] {
			return fmt.Errorf("invalid status value")
		}
	}

	// Type is optional, but if provided must be valid
	if r.Type != "" {
		validTypes := map[string]bool{
			"POST":    true,
			"ARTICLE": true,
		}
		if !validTypes[strings.ToUpper(r.Type)] {
			return fmt.Errorf("invalid type value")
		}
	}

	if r.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}

	if r.Limit > 1000 {
		return fmt.Errorf("limit exceeds maximum of 1000")
	}

	return nil
}

// isValidObjectID validates MongoDB ObjectID format
func isValidObjectID(id string) bool {
	if id == "" {
		return false
	}
	return objectIDRegex.MatchString(id)
}
