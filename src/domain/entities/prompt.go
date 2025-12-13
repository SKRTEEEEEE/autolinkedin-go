package entities

import (
	"fmt"
	"strings"
	"time"
)

// PromptType represents the type of prompt
type PromptType string

const (
	PromptTypeIdeas  PromptType = "ideas"
	PromptTypeDrafts PromptType = "drafts"
)

// Prompt represents a template for LLM prompts
type Prompt struct {
	ID             string
	UserID         string
	Type           PromptType
	Name           string // Unique identifier for the prompt
	StyleName      string // For backward compatibility
	PromptTemplate string
	Active         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const (
	MinPromptTemplateLength = 10
	MaxPromptTemplateLength = 5000
	MaxStyleNameLength      = 50
	MaxNameLength           = 50
)

// Validate validates the prompt entity
func (p *Prompt) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("prompt ID cannot be empty")
	}

	if p.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if err := p.ValidateType(); err != nil {
		return err
	}

	if err := p.ValidateName(); err != nil {
		return err
	}

	if err := p.ValidateStyleName(); err != nil {
		return err
	}

	if err := p.ValidateTemplate(); err != nil {
		return err
	}

	if p.CreatedAt.IsZero() {
		return fmt.Errorf("created timestamp cannot be zero")
	}

	if p.UpdatedAt.IsZero() {
		return fmt.Errorf("updated timestamp cannot be zero")
	}

	if p.CreatedAt.After(time.Now()) {
		return fmt.Errorf("created timestamp cannot be in the future")
	}

	if p.UpdatedAt.Before(p.CreatedAt) {
		return fmt.Errorf("updated timestamp cannot be before created timestamp")
	}

	return nil
}

// ValidateType validates the prompt type
func (p *Prompt) ValidateType() error {
	if p.Type != PromptTypeIdeas && p.Type != PromptTypeDrafts {
		return fmt.Errorf("invalid prompt type: must be '%s' or '%s'", PromptTypeIdeas, PromptTypeDrafts)
	}
	return nil
}

// ValidateName validates the prompt name
func (p *Prompt) ValidateName() error {
	if p.Name == "" {
		return fmt.Errorf("prompt name is required")
	}

	trimmed := strings.TrimSpace(p.Name)
	if trimmed == "" {
		return fmt.Errorf("prompt name cannot be only whitespace")
	}

	if len(p.Name) > MaxNameLength {
		return fmt.Errorf("prompt name too long (maximum %d characters)", MaxNameLength)
	}

	return nil
}

// ValidateStyleName validates the style name
func (p *Prompt) ValidateStyleName() error {
	// Style name is optional now, required only for backward compatibility
	if p.StyleName != "" {
		trimmed := strings.TrimSpace(p.StyleName)
		if trimmed == "" {
			return fmt.Errorf("style name cannot be only whitespace")
		}

		if len(p.StyleName) > MaxStyleNameLength {
			return fmt.Errorf("style name too long (maximum %d characters)", MaxStyleNameLength)
		}
	}

	return nil
}

// ValidateTemplate validates the prompt template
func (p *Prompt) ValidateTemplate() error {
	if p.PromptTemplate == "" {
		return fmt.Errorf("prompt template cannot be empty")
	}

	trimmed := strings.TrimSpace(p.PromptTemplate)
	if trimmed == "" {
		return fmt.Errorf("prompt template cannot be only whitespace")
	}

	if len(trimmed) < MinPromptTemplateLength {
		return fmt.Errorf("prompt template too short (minimum %d characters)", MinPromptTemplateLength)
	}

	if len(p.PromptTemplate) > MaxPromptTemplateLength {
		return fmt.Errorf("prompt template too long (maximum %d characters)", MaxPromptTemplateLength)
	}

	return nil
}

// IsOwnedBy checks if prompt belongs to specified user
func (p *Prompt) IsOwnedBy(userID string) bool {
	return p.UserID != "" && p.UserID == userID
}

// IsIdeasPrompt checks if this is an ideas generation prompt
func (p *Prompt) IsIdeasPrompt() bool {
	return p.Type == PromptTypeIdeas
}

// IsDraftsPrompt checks if this is a drafts generation prompt
func (p *Prompt) IsDraftsPrompt() bool {
	return p.Type == PromptTypeDrafts
}

// UpdateTemplate updates the prompt template and timestamp
func (p *Prompt) UpdateTemplate(template string) error {
	// Temporarily set template to validate
	oldTemplate := p.PromptTemplate
	p.PromptTemplate = template

	if err := p.ValidateTemplate(); err != nil {
		p.PromptTemplate = oldTemplate
		return err
	}

	p.UpdatedAt = time.Now()
	return nil
}

// Deactivate marks the prompt as inactive
func (p *Prompt) Deactivate() {
	p.Active = false
	p.UpdatedAt = time.Now()
}

// Activate marks the prompt as active
func (p *Prompt) Activate() {
	p.Active = true
	p.UpdatedAt = time.Now()
}
