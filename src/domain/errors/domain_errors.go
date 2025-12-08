package errors

import "fmt"

// ErrInvalidUserCredentials represents invalid user credentials error
type ErrInvalidUserCredentials struct {
	UserID string
	Reason string
}

func (e *ErrInvalidUserCredentials) Error() string {
	return fmt.Sprintf("invalid user credentials for user %s: %s", e.UserID, e.Reason)
}

// NewInvalidUserCredentials creates a new invalid credentials error
func NewInvalidUserCredentials(userID, reason string) *ErrInvalidUserCredentials {
	return &ErrInvalidUserCredentials{
		UserID: userID,
		Reason: reason,
	}
}

// ErrTopicNotFound represents topic not found error
type ErrTopicNotFound struct {
	TopicID string
}

func (e *ErrTopicNotFound) Error() string {
	return fmt.Sprintf("topic not found: %s", e.TopicID)
}

// NewTopicNotFound creates a new topic not found error
func NewTopicNotFound(topicID string) *ErrTopicNotFound {
	return &ErrTopicNotFound{TopicID: topicID}
}

// ErrIdeaExpired represents expired idea error
type ErrIdeaExpired struct {
	IdeaID string
}

func (e *ErrIdeaExpired) Error() string {
	return fmt.Sprintf("idea has expired: %s", e.IdeaID)
}

// NewIdeaExpired creates a new idea expired error
func NewIdeaExpired(ideaID string) *ErrIdeaExpired {
	return &ErrIdeaExpired{IdeaID: ideaID}
}

// ErrIdeaNotFound represents idea not found error
type ErrIdeaNotFound struct {
	IdeaID string
}

func (e *ErrIdeaNotFound) Error() string {
	return fmt.Sprintf("idea not found: %s", e.IdeaID)
}

// NewIdeaNotFound creates a new idea not found error
func NewIdeaNotFound(ideaID string) *ErrIdeaNotFound {
	return &ErrIdeaNotFound{IdeaID: ideaID}
}

// ErrDraftAlreadyPublished represents already published draft error
type ErrDraftAlreadyPublished struct {
	DraftID        string
	LinkedInPostID string
}

func (e *ErrDraftAlreadyPublished) Error() string {
	return fmt.Sprintf("draft %s is already published (LinkedIn post ID: %s)", e.DraftID, e.LinkedInPostID)
}

// NewDraftAlreadyPublished creates a new draft already published error
func NewDraftAlreadyPublished(draftID, linkedInPostID string) *ErrDraftAlreadyPublished {
	return &ErrDraftAlreadyPublished{
		DraftID:        draftID,
		LinkedInPostID: linkedInPostID,
	}
}

// ErrDraftNotFound represents draft not found error
type ErrDraftNotFound struct {
	DraftID string
}

func (e *ErrDraftNotFound) Error() string {
	return fmt.Sprintf("draft not found: %s", e.DraftID)
}

// NewDraftNotFound creates a new draft not found error
func NewDraftNotFound(draftID string) *ErrDraftNotFound {
	return &ErrDraftNotFound{DraftID: draftID}
}

// ErrRefinementLimitExceeded represents refinement limit exceeded error
type ErrRefinementLimitExceeded struct {
	DraftID      string
	CurrentCount int
	MaxLimit     int
}

func (e *ErrRefinementLimitExceeded) Error() string {
	return fmt.Sprintf("refinement limit exceeded for draft %s: %d/%d", e.DraftID, e.CurrentCount, e.MaxLimit)
}

// NewRefinementLimitExceeded creates a new refinement limit exceeded error
func NewRefinementLimitExceeded(draftID string, currentCount, maxLimit int) *ErrRefinementLimitExceeded {
	return &ErrRefinementLimitExceeded{
		DraftID:      draftID,
		CurrentCount: currentCount,
		MaxLimit:     maxLimit,
	}
}

// ErrInvalidDraftType represents invalid draft type error
type ErrInvalidDraftType struct {
	Type string
}

func (e *ErrInvalidDraftType) Error() string {
	return fmt.Sprintf("invalid draft type: %s", e.Type)
}

// NewInvalidDraftType creates a new invalid draft type error
func NewInvalidDraftType(draftType string) *ErrInvalidDraftType {
	return &ErrInvalidDraftType{Type: draftType}
}

// ErrInvalidDraftStatus represents invalid draft status error
type ErrInvalidDraftStatus struct {
	Status string
}

func (e *ErrInvalidDraftStatus) Error() string {
	return fmt.Sprintf("invalid draft status: %s", e.Status)
}

// NewInvalidDraftStatus creates a new invalid draft status error
func NewInvalidDraftStatus(status string) *ErrInvalidDraftStatus {
	return &ErrInvalidDraftStatus{Status: status}
}

// ErrUnauthorizedAccess represents unauthorized access error
type ErrUnauthorizedAccess struct {
	UserID     string
	ResourceID string
	Resource   string
}

func (e *ErrUnauthorizedAccess) Error() string {
	return fmt.Sprintf("user %s is not authorized to access %s: %s", e.UserID, e.Resource, e.ResourceID)
}

// NewUnauthorizedAccess creates a new unauthorized access error
func NewUnauthorizedAccess(userID, resource, resourceID string) *ErrUnauthorizedAccess {
	return &ErrUnauthorizedAccess{
		UserID:     userID,
		Resource:   resource,
		ResourceID: resourceID,
	}
}

// ErrInvalidEmail represents invalid email format error
type ErrInvalidEmail struct {
	Email string
}

func (e *ErrInvalidEmail) Error() string {
	return fmt.Sprintf("invalid email format: %s", e.Email)
}

// NewInvalidEmail creates a new invalid email error
func NewInvalidEmail(email string) *ErrInvalidEmail {
	return &ErrInvalidEmail{Email: email}
}

// ErrInvalidTransition represents invalid status transition error
type ErrInvalidTransition struct {
	From     string
	To       string
	Resource string
}

func (e *ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid %s transition from %s to %s", e.Resource, e.From, e.To)
}

// NewInvalidTransition creates a new invalid transition error
func NewInvalidTransition(resource, from, to string) *ErrInvalidTransition {
	return &ErrInvalidTransition{
		Resource: resource,
		From:     from,
		To:       to,
	}
}

// ErrValidation represents a generic validation error
type ErrValidation struct {
	Field   string
	Message string
}

func (e *ErrValidation) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ErrValidation {
	return &ErrValidation{
		Field:   field,
		Message: message,
	}
}

// LLMResponseError represents an invalid or unparsable response from the LLM provider
type LLMResponseError struct {
	Operation   string
	Reason      string
	Prompt      string
	RawResponse string
	Err         error
}

// Error implements the error interface
func (e *LLMResponseError) Error() string {
	base := "llm response error"
	if e != nil {
		if e.Operation != "" {
			base = fmt.Sprintf("%s (%s)", base, e.Operation)
		}
		if e.Reason != "" {
			base = fmt.Sprintf("%s: %s", base, e.Reason)
		}
		if e.Err != nil {
			return fmt.Sprintf("%s: %v", base, e.Err)
		}
	}
	return base
}

// Unwrap returns the underlying error
func (e *LLMResponseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewLLMResponseError creates a new LLM response error
func NewLLMResponseError(operation, reason, prompt, raw string, err error) *LLMResponseError {
	return &LLMResponseError{
		Operation:   operation,
		Reason:      reason,
		Prompt:      prompt,
		RawResponse: raw,
		Err:         err,
	}
}
