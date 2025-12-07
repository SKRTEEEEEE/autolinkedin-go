package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// PromptsHandler handles prompt-related HTTP requests
type PromptsHandler struct {
	promptsRepo interfaces.PromptsRepository
	userRepo    interfaces.UserRepository
	logger      *zap.Logger
}

// NewPromptsHandler creates a new PromptsHandler instance
func NewPromptsHandler(
	promptsRepo interfaces.PromptsRepository,
	userRepo interfaces.UserRepository,
	logger *zap.Logger,
) *PromptsHandler {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &PromptsHandler{
		promptsRepo: promptsRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// PromptDTO represents a prompt in the response
type PromptDTO struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	Type           string `json:"type"`
	StyleName      string `json:"style_name,omitempty"`
	PromptTemplate string `json:"prompt_template"`
	Active         bool   `json:"active"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// CreatePromptRequest represents the request to create a prompt
type CreatePromptRequest struct {
	UserID         string `json:"user_id"`
	Type           string `json:"type"`
	StyleName      string `json:"style_name,omitempty"`
	PromptTemplate string `json:"prompt_template"`
}

// Validate validates the create prompt request
func (r *CreatePromptRequest) Validate() error {
	if r.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if r.Type == "" {
		return fmt.Errorf("type is required")
	}
	if r.Type != string(entities.PromptTypeIdeas) && r.Type != string(entities.PromptTypeDrafts) {
		return fmt.Errorf("type must be 'ideas' or 'drafts'")
	}
	if r.Type == string(entities.PromptTypeDrafts) && r.StyleName == "" {
		return fmt.Errorf("style_name is required for drafts prompts")
	}
	if r.PromptTemplate == "" {
		return fmt.Errorf("prompt_template is required")
	}
	return nil
}

// UpdatePromptRequest represents the request to update a prompt
type UpdatePromptRequest struct {
	PromptTemplate *string `json:"prompt_template,omitempty"`
	Active         *bool   `json:"active,omitempty"`
}

// Validate validates the update prompt request
func (r *UpdatePromptRequest) Validate() error {
	if r.PromptTemplate == nil && r.Active == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}
	return nil
}

// ListPrompts handles GET /v1/prompts/{userId}
func (h *PromptsHandler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	userID := vars["userId"]

	if userID == "" {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, "user_id is required", nil, h.logger)
		return
	}

	// Verify user exists
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}
	if user == nil {
		WriteError(w, http.StatusNotFound, ErrorCodeNotFound, "user not found", nil, h.logger)
		return
	}

	// Get query parameters
	promptType := r.URL.Query().Get("type")

	var prompts []*entities.Prompt

	if promptType != "" {
		// Validate type
		if promptType != string(entities.PromptTypeIdeas) && promptType != string(entities.PromptTypeDrafts) {
			WriteError(w, http.StatusBadRequest, ErrorCodeValidation, "invalid type parameter", nil, h.logger)
			return
		}

		prompts, err = h.promptsRepo.ListByUserIDAndType(ctx, userID, entities.PromptType(promptType))
	} else {
		prompts, err = h.promptsRepo.ListByUserID(ctx, userID)
	}

	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	// Convert to DTOs
	promptDTOs := make([]PromptDTO, 0, len(prompts))
	for _, prompt := range prompts {
		promptDTOs = append(promptDTOs, PromptDTO{
			ID:             prompt.ID,
			UserID:         prompt.UserID,
			Type:           string(prompt.Type),
			StyleName:      prompt.StyleName,
			PromptTemplate: prompt.PromptTemplate,
			Active:         prompt.Active,
			CreatedAt:      prompt.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      prompt.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"prompts": promptDTOs,
		"count":   len(promptDTOs),
	}, h.logger)
}

// CreatePrompt handles POST /v1/prompts
func (h *PromptsHandler) CreatePrompt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeInvalidInput, "Invalid request body", nil, h.logger)
		return
	}
	defer r.Body.Close()

	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
		return
	}

	// Verify user exists
	user, err := h.userRepo.FindByID(ctx, req.UserID)
	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}
	if user == nil {
		WriteError(w, http.StatusNotFound, ErrorCodeNotFound, "user not found", nil, h.logger)
		return
	}

	now := time.Now()
	prompt := &entities.Prompt{
		ID:             primitive.NewObjectID().Hex(),
		UserID:         req.UserID,
		Type:           entities.PromptType(req.Type),
		StyleName:      req.StyleName,
		PromptTemplate: req.PromptTemplate,
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := prompt.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
		return
	}

	promptID, err := h.promptsRepo.Create(ctx, prompt)
	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	WriteJSON(w, http.StatusCreated, PromptDTO{
		ID:             promptID,
		UserID:         prompt.UserID,
		Type:           string(prompt.Type),
		StyleName:      prompt.StyleName,
		PromptTemplate: prompt.PromptTemplate,
		Active:         prompt.Active,
		CreatedAt:      prompt.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      prompt.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, h.logger)
}

// UpdatePrompt handles PATCH /v1/prompts/{promptId}
func (h *PromptsHandler) UpdatePrompt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	promptID := vars["promptId"]

	if promptID == "" {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, "prompt_id is required", nil, h.logger)
		return
	}

	var req UpdatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeInvalidInput, "Invalid request body", nil, h.logger)
		return
	}
	defer r.Body.Close()

	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
		return
	}

	prompt, err := h.promptsRepo.FindByID(ctx, promptID)
	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}
	if prompt == nil {
		WriteError(w, http.StatusNotFound, ErrorCodeNotFound, "prompt not found", nil, h.logger)
		return
	}

	// Update fields
	if req.PromptTemplate != nil {
		if err := prompt.UpdateTemplate(*req.PromptTemplate); err != nil {
			WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
			return
		}
	}

	if req.Active != nil {
		if *req.Active {
			prompt.Activate()
		} else {
			prompt.Deactivate()
		}
	}

	if err := h.promptsRepo.Update(ctx, prompt); err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	WriteJSON(w, http.StatusOK, PromptDTO{
		ID:             prompt.ID,
		UserID:         prompt.UserID,
		Type:           string(prompt.Type),
		StyleName:      prompt.StyleName,
		PromptTemplate: prompt.PromptTemplate,
		Active:         prompt.Active,
		CreatedAt:      prompt.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      prompt.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, h.logger)
}

// RegisterRoutes registers all prompt routes
func (h *PromptsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/v1/prompts/{userId}", h.ListPrompts).Methods(http.MethodGet)
	router.HandleFunc("/v1/prompts", h.CreatePrompt).Methods(http.MethodPost)
	router.HandleFunc("/v1/prompts/{promptId}", h.UpdatePrompt).Methods(http.MethodPatch)
}
