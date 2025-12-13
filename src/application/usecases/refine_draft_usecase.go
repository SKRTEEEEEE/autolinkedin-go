package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/linkgen-ai/backend/src/domain/entities"
	domainErrors "github.com/linkgen-ai/backend/src/domain/errors"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/infrastructure/database"
)

// RefineDraftUseCase orchestrates draft refinement with user feedback
type RefineDraftUseCase struct {
	draftRepo  interfaces.DraftRepository
	llmService interfaces.LLMService
}

// NewRefineDraftUseCase creates a new instance of RefineDraftUseCase
func NewRefineDraftUseCase(
	draftRepo interfaces.DraftRepository,
	llmService interfaces.LLMService,
) *RefineDraftUseCase {
	return &RefineDraftUseCase{
		draftRepo:  draftRepo,
		llmService: llmService,
	}
}

// RefineDraftInput represents input for draft refinement
type RefineDraftInput struct {
	DraftID    string
	UserPrompt string
}

// Execute refines a draft based on user feedback
func (uc *RefineDraftUseCase) Execute(ctx context.Context, input RefineDraftInput) (*entities.Draft, error) {
	normalized := RefineDraftInput{
		DraftID:    strings.TrimSpace(input.DraftID),
		UserPrompt: strings.TrimSpace(input.UserPrompt),
	}

	// Validate input
	if err := uc.validateInput(normalized); err != nil {
		return nil, err
	}
	input = normalized

	// Get draft from repository
	draft, err := uc.draftRepo.FindByID(ctx, input.DraftID)
	if err != nil {
		if errors.Is(err, database.ErrEntityNotFound) {
			return nil, domainErrors.NewDraftNotFound(input.DraftID)
		}
		if errors.Is(err, database.ErrInvalidID) {
			return nil, domainErrors.NewValidationError("draft_id", "invalid draft ID")
		}
		return nil, fmt.Errorf("failed to retrieve draft: %w", err)
	}
	if draft == nil {
		return nil, domainErrors.NewDraftNotFound(input.DraftID)
	}

	// Verify draft can be refined
	if !draft.CanBeRefined() {
		return nil, domainErrors.NewInvalidDraftStatus(string(draft.Status))
	}

	// Check refinement limit
	if len(draft.RefinementHistory) >= entities.MaxRefinements {
		return nil, domainErrors.NewRefinementLimitExceeded(
			draft.ID,
			len(draft.RefinementHistory),
			entities.MaxRefinements,
		)
	}

	// Build conversation history from refinement history
	history := uc.buildConversationHistory(draft)

	// Call LLM to refine content
	refinedContent, err := uc.llmService.RefineDraft(ctx, draft.Content, input.UserPrompt, history)
	if err != nil {
		return nil, fmt.Errorf("LLM service error: %w", err)
	}

	// Validate refined content
	trimmedContent := strings.TrimSpace(refinedContent)
	if trimmedContent == "" {
		return nil, domainErrors.NewValidationError("refined_content", "LLM returned empty refined content")
	}

	// Add refinement to draft
	if err := draft.AddRefinement(trimmedContent, input.UserPrompt); err != nil {
		return nil, domainErrors.NewValidationError("refinement", err.Error())
	}

	// Validate the refined draft
	if err := draft.Validate(); err != nil {
		return nil, domainErrors.NewValidationError("draft", err.Error())
	}

	// Save updated draft
	if err := uc.saveDraft(ctx, draft); err != nil {
		return nil, err
	}

	return draft, nil
}
func (uc *RefineDraftUseCase) validateInput(input RefineDraftInput) error {
	if input.DraftID == "" && input.UserPrompt == "" {
		return domainErrors.NewValidationError("draft_id", "draft ID and prompt cannot be empty")
	}

	if input.DraftID == "" {
		return domainErrors.NewValidationError("draft_id", "draft ID cannot be empty")
	}

	if input.UserPrompt == "" {
		return domainErrors.NewValidationError("prompt", "user prompt cannot be empty")
	}

	if len([]rune(input.UserPrompt)) < 10 {
		return domainErrors.NewValidationError("prompt", "prompt must be at least 10 characters")
	}

	if len([]rune(input.UserPrompt)) > 500 {
		return domainErrors.NewValidationError("prompt", "prompt exceeds maximum of 500 characters")
	}

	return nil
}

// buildConversationHistory creates a history array from refinement entries
func (uc *RefineDraftUseCase) buildConversationHistory(draft *entities.Draft) []string {
	if len(draft.RefinementHistory) == 0 {
		return []string{}
	}

	history := make([]string, 0, len(draft.RefinementHistory)*2)

	for _, entry := range draft.RefinementHistory {
		history = append(history, fmt.Sprintf("Versión %d - Prompt del usuario: %s", entry.Version, entry.Prompt))
		history = append(history, fmt.Sprintf("Versión %d - Respuesta generada: %s", entry.Version, entry.Content))
	}

	return history
}

// saveDraft saves the updated draft to repository
func (uc *RefineDraftUseCase) saveDraft(ctx context.Context, draft *entities.Draft) error {
	// Update draft with refinement entry
	updates := map[string]interface{}{
		"content":            draft.Content,
		"status":             draft.Status,
		"refinement_history": draft.RefinementHistory,
		"updated_at":         draft.UpdatedAt,
	}

	if err := uc.draftRepo.Update(ctx, draft.ID, updates); err != nil {
		if errors.Is(err, database.ErrEntityNotFound) {
			return domainErrors.NewDraftNotFound(draft.ID)
		}
		return fmt.Errorf("failed to save draft: %w", err)
	}

	return nil
}
