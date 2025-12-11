package entities

import (
	"fmt"
	"strings"
	"time"
)

// DraftType represents the type of draft content
type DraftType string

const (
	DraftTypePost    DraftType = "POST"
	DraftTypeArticle DraftType = "ARTICLE"
)

// DraftStatus represents the current state of a draft
type DraftStatus string

const (
	DraftStatusDraft     DraftStatus = "DRAFT"
	DraftStatusRefined   DraftStatus = "REFINED"
	DraftStatusPublished DraftStatus = "PUBLISHED"
	DraftStatusFailed    DraftStatus = "FAILED"
)

// RefinementEntry represents a single refinement in the history
type RefinementEntry struct {
	Timestamp time.Time
	Prompt    string
	Content   string
	Version   int
}

// Draft represents a content draft ready for publication
type Draft struct {
	ID                string
	UserID            string
	IdeaID            *string
	Type              DraftType
	Title             string
	Content           string
	Status            DraftStatus
	RefinementHistory []RefinementEntry
	PublishedAt       *time.Time
	LinkedInPostID    string
	Metadata          map[string]interface{}
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

const (
	MinPostContentLength    = 10
	MaxPostContentLength    = 3000
	MinArticleContentLength = 100
	MaxArticleContentLength = 110000
	MinArticleTitleLength   = 5
	MaxArticleTitleLength   = 200
	MaxRefinements          = 10
)

// Validate validates the draft entity
func (d *Draft) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("draft ID cannot be empty")
	}

	if d.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if !d.isValidType() {
		return fmt.Errorf("invalid draft type: %s", d.Type)
	}

	if !d.isValidStatus() {
		return fmt.Errorf("invalid draft status: %s", d.Status)
	}

	if err := d.ValidateForType(); err != nil {
		return err
	}

	return nil
}

// isValidType checks if draft type is valid
func (d *Draft) isValidType() bool {
	return d.Type == DraftTypePost || d.Type == DraftTypeArticle
}

// isValidStatus checks if draft status is valid
func (d *Draft) isValidStatus() bool {
	return d.Status == DraftStatusDraft ||
		d.Status == DraftStatusRefined ||
		d.Status == DraftStatusPublished ||
		d.Status == DraftStatusFailed
}

// ValidateForType performs type-specific validation
func (d *Draft) ValidateForType() error {
	trimmedContent := strings.TrimSpace(d.Content)

	if trimmedContent == "" {
		return fmt.Errorf("draft content cannot be empty")
	}

	switch d.Type {
	case DraftTypePost:
		return d.validatePost(trimmedContent)
	case DraftTypeArticle:
		return d.validateArticle(trimmedContent)
	default:
		return fmt.Errorf("unknown draft type: %s", d.Type)
	}
}

// validatePost validates post-specific rules
func (d *Draft) validatePost(content string) error {
	if len(content) < MinPostContentLength {
		return fmt.Errorf("post content too short (minimum %d characters)", MinPostContentLength)
	}

	if len(content) > MaxPostContentLength {
		return fmt.Errorf("post content too long (maximum %d characters)", MaxPostContentLength)
	}

	return nil
}

// validateArticle validates article-specific rules
func (d *Draft) validateArticle(content string) error {
	trimmedTitle := strings.TrimSpace(d.Title)

	if trimmedTitle == "" {
		return fmt.Errorf("article title cannot be empty")
	}

	if len(trimmedTitle) < MinArticleTitleLength {
		return fmt.Errorf("article title too short (minimum %d characters)", MinArticleTitleLength)
	}

	if len(trimmedTitle) > MaxArticleTitleLength {
		return fmt.Errorf("article title too long (maximum %d characters)", MaxArticleTitleLength)
	}

	if len(content) < MinArticleContentLength {
		return fmt.Errorf("article content too short (minimum %d characters)", MinArticleContentLength)
	}

	if len(content) > MaxArticleContentLength {
		return fmt.Errorf("article content too long (maximum %d characters)", MaxArticleContentLength)
	}

	return nil
}

// CanBeRefined checks if draft can be refined
func (d *Draft) CanBeRefined() bool {
	return d.Status == DraftStatusDraft || d.Status == DraftStatusRefined
}

// CanBePublished checks if draft is ready for publishing
func (d *Draft) CanBePublished() error {
	if d.Status == DraftStatusPublished {
		return fmt.Errorf("draft is already published")
	}

	if d.Status == DraftStatusFailed {
		return fmt.Errorf("cannot publish failed draft")
	}

	if err := d.ValidateForType(); err != nil {
		return fmt.Errorf("draft validation failed: %w", err)
	}

	return nil
}

// AddRefinement adds a refinement to the history
func (d *Draft) AddRefinement(content, prompt string) error {
	if !d.CanBeRefined() {
		return fmt.Errorf("draft cannot be refined in current status: %s", d.Status)
	}

	if len(d.RefinementHistory) >= MaxRefinements {
		return fmt.Errorf("refinement limit exceeded (maximum %d)", MaxRefinements)
	}

	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return fmt.Errorf("refinement content cannot be empty")
	}

	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "" {
		return fmt.Errorf("refinement prompt cannot be empty")
	}

	entry := RefinementEntry{
		Timestamp: time.Now(),
		Prompt:    trimmedPrompt,
		Content:   trimmedContent,
		Version:   len(d.RefinementHistory) + 1,
	}

	d.RefinementHistory = append(d.RefinementHistory, entry)
	d.Content = trimmedContent
	d.Status = DraftStatusRefined
	d.UpdatedAt = time.Now()

	return nil
}

// MarkAsPublished marks draft as published with LinkedIn post ID
func (d *Draft) MarkAsPublished(linkedInID string) error {
	if d.Status == DraftStatusPublished {
		return fmt.Errorf("draft is already published")
	}

	trimmedID := strings.TrimSpace(linkedInID)
	if trimmedID == "" {
		return fmt.Errorf("LinkedIn post ID cannot be empty")
	}

	now := time.Now()
	d.Status = DraftStatusPublished
	d.LinkedInPostID = trimmedID
	d.PublishedAt = &now
	d.UpdatedAt = now

	return nil
}

// MarkAsFailed marks draft as failed
func (d *Draft) MarkAsFailed() {
	d.Status = DraftStatusFailed
	d.UpdatedAt = time.Now()
}

// GetLatestVersion returns the latest content version
func (d *Draft) GetLatestVersion() string {
	if len(d.RefinementHistory) == 0 {
		return d.Content
	}

	return d.RefinementHistory[len(d.RefinementHistory)-1].Content
}

// CanTransitionTo checks if status transition is allowed
func (d *Draft) CanTransitionTo(newStatus DraftStatus) error {
	// Define allowed transitions
	allowedTransitions := map[DraftStatus][]DraftStatus{
		DraftStatusDraft: {
			DraftStatusRefined,
			DraftStatusPublished,
			DraftStatusFailed,
		},
		DraftStatusRefined: {
			DraftStatusDraft,
			DraftStatusPublished,
			DraftStatusFailed,
		},
		DraftStatusPublished: {
			// No transitions allowed from published
		},
		DraftStatusFailed: {
			DraftStatusDraft, // Allow retry
		},
	}

	allowed, exists := allowedTransitions[d.Status]
	if !exists {
		return fmt.Errorf("unknown current status: %s", d.Status)
	}

	for _, allowedStatus := range allowed {
		if allowedStatus == newStatus {
			return nil
		}
	}

	return fmt.Errorf("transition from %s to %s is not allowed", d.Status, newStatus)
}
