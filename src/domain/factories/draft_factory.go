package factories

import (
	"fmt"
	"time"

	"github.com/linkgen-ai/backend/domain/entities"
)

// NewDraft creates a new Draft entity with type-specific setup
func NewDraft(id, userID string, draftType entities.DraftType, content string) (*entities.Draft, error) {
	now := time.Now()
	
	draft := &entities.Draft{
		ID:                id,
		UserID:            userID,
		IdeaID:            nil,
		Type:              draftType,
		Title:             "",
		Content:           content,
		Status:            entities.DraftStatusDraft,
		RefinementHistory: []entities.RefinementEntry{},
		PublishedAt:       nil,
		LinkedInPostID:    "",
		Metadata:          make(map[string]interface{}),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := draft.Validate(); err != nil {
		return nil, fmt.Errorf("draft validation failed: %w", err)
	}

	return draft, nil
}

// NewPostDraft creates a new POST type draft
func NewPostDraft(id, userID, content string) (*entities.Draft, error) {
	return NewDraft(id, userID, entities.DraftTypePost, content)
}

// NewArticleDraft creates a new ARTICLE type draft with title
func NewArticleDraft(id, userID, title, content string) (*entities.Draft, error) {
	draft, err := NewDraft(id, userID, entities.DraftTypeArticle, content)
	if err != nil {
		return nil, err
	}

	draft.Title = title

	if err := draft.Validate(); err != nil {
		return nil, fmt.Errorf("article draft validation failed: %w", err)
	}

	return draft, nil
}

// NewDraftFromIdea creates a new Draft from an Idea
func NewDraftFromIdea(id, userID string, ideaID string, draftType entities.DraftType, content string) (*entities.Draft, error) {
	draft, err := NewDraft(id, userID, draftType, content)
	if err != nil {
		return nil, err
	}

	draft.IdeaID = &ideaID

	return draft, nil
}

// NewPostDraftFromIdea creates a POST draft from an Idea
func NewPostDraftFromIdea(id, userID, ideaID, content string) (*entities.Draft, error) {
	return NewDraftFromIdea(id, userID, ideaID, entities.DraftTypePost, content)
}

// NewArticleDraftFromIdea creates an ARTICLE draft from an Idea
func NewArticleDraftFromIdea(id, userID, ideaID, title, content string) (*entities.Draft, error) {
	draft, err := NewDraftFromIdea(id, userID, ideaID, entities.DraftTypeArticle, content)
	if err != nil {
		return nil, err
	}

	draft.Title = title

	if err := draft.Validate(); err != nil {
		return nil, fmt.Errorf("article draft validation failed: %w", err)
	}

	return draft, nil
}

// NewDraftWithMetadata creates a new Draft with metadata
func NewDraftWithMetadata(id, userID string, draftType entities.DraftType, content string, metadata map[string]interface{}) (*entities.Draft, error) {
	draft, err := NewDraft(id, userID, draftType, content)
	if err != nil {
		return nil, err
	}

	if metadata != nil {
		draft.Metadata = metadata
	}

	return draft, nil
}
