package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
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
	// Validate input
	if err := uc.validateInput(input); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Get draft from repository
	draft, err := uc.draftRepo.FindByID(ctx, input.DraftID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve draft: %w", err)
	}
	if draft == nil {
		return nil, fmt.Errorf("draft not found: %s", input.DraftID)
	}

	// Verify draft can be refined
	if !draft.CanBeRefined() {
		return nil, fmt.Errorf("draft cannot be refined in current status: %s", draft.Status)
	}

	// Check refinement limit
	if len(draft.RefinementHistory) >= entities.MaxRefinements {
		return nil, fmt.Errorf("refinement limit exceeded (maximum %d)", entities.MaxRefinements)
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
		return nil, fmt.Errorf("LLM returned empty refined content")
	}

	// Add refinement to draft
	if err := draft.AddRefinement(trimmedContent, input.UserPrompt); err != nil {
		return nil, fmt.Errorf("failed to add refinement: %w", err)
	}

	// Validate the refined draft
	if err := draft.Validate(); err != nil {
		return nil, fmt.Errorf("refined draft validation failed: %w", err)
	}

	// Save updated draft
	if err := uc.saveDraft(ctx, draft); err != nil {
		return nil, err
	}

	return draft, nil
}

// validateInput validates the input parameters
func (uc *RefineDraftUseCase) validateInput(input RefineDraftInput) error {
	draftID := strings.TrimSpace(input.DraftID)
	userPrompt := strings.TrimSpace(input.UserPrompt)

	if draftID == "" && userPrompt == "" {
		return fmt.Errorf("draft ID and prompt cannot be empty")
	}

	if draftID == "" {
		return fmt.Errorf("draft ID cannot be empty")
	}

	if userPrompt == "" {
		return fmt.Errorf("user prompt cannot be empty")
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
		history = append(history, fmt.Sprintf("User: %s", entry.Prompt))
		history = append(history, fmt.Sprintf("Assistant: %s", entry.Content))
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
		return fmt.Errorf("failed to save draft: %w", err)
	}

	return nil
}
