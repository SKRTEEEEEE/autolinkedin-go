package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/linkgen-ai/backend/src/application/usecases"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/domain/valueobjects"
	"github.com/linkgen-ai/backend/src/infrastructure/messaging/nats"
	"go.uber.org/zap"
)

// DraftsHandler handles draft-related HTTP requests
type DraftsHandler struct {
	refineDraftUseCase *usecases.RefineDraftUseCase
	draftRepository    interfaces.DraftRepository
	natsPublisher      *nats.Publisher
	logger             *zap.Logger
}

// NewDraftsHandler creates a new DraftsHandler instance
func NewDraftsHandler(
	refineDraftUseCase *usecases.RefineDraftUseCase,
	draftRepository interfaces.DraftRepository,
	natsPublisher *nats.Publisher,
	logger *zap.Logger,
) *DraftsHandler {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &DraftsHandler{
		refineDraftUseCase: refineDraftUseCase,
		draftRepository:    draftRepository,
		natsPublisher:      natsPublisher,
		logger:             logger,
	}
}

// DraftGenerationMessage represents the message queued to NATS
type DraftGenerationMessage struct {
	JobID      string    `json:"job_id"`
	UserID     string    `json:"user_id"`
	IdeaID     string    `json:"idea_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	RetryCount int       `json:"retry_count"`
}

// GenerateDraftsResponse represents the response for draft generation request
type GenerateDraftsResponse struct {
	Message string `json:"message"`
	JobID   string `json:"job_id"`
}

// GenerateDrafts handles POST /v1/drafts/generate
func (h *DraftsHandler) GenerateDrafts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req GenerateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeInvalidInput, "Invalid request body", nil, h.logger)
		return
	}
	defer r.Body.Close()

	// Validate request
	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
		return
	}

	// Generate job ID
	jobID := uuid.New().String()

	// Create message for NATS
	message := DraftGenerationMessage{
		JobID:      jobID,
		UserID:     req.UserID,
		IdeaID:     req.IdeaID,
		Timestamp:  time.Now(),
		RetryCount: 0,
	}

	// Publish to NATS queue
	if err := h.natsPublisher.Publish(ctx, message); err != nil {
		h.logger.Error("failed to queue draft generation",
			zap.String("job_id", jobID),
			zap.Error(err),
		)
		WriteError(w, http.StatusServiceUnavailable, ErrorCodeServiceTimeout, "Failed to queue draft generation", nil, h.logger)
		return
	}

	h.logger.Info("draft generation queued",
		zap.String("job_id", jobID),
		zap.String("user_id", req.UserID),
		zap.String("idea_id", req.IdeaID),
	)

	// Return 202 Accepted
	response := GenerateDraftsResponse{
		Message: "Draft generation started",
		JobID:   jobID,
	}

	WriteJSON(w, http.StatusAccepted, response, h.logger)
}

// GetDraftsResponse represents the response for listing drafts
type GetDraftsResponse struct {
	Drafts []DraftDTO `json:"drafts"`
	Count  int        `json:"count"`
}

// DraftDTO represents a draft in the response
type DraftDTO struct {
	ID                string                 `json:"id"`
	UserID            string                 `json:"user_id"`
	IdeaID            *string                `json:"idea_id,omitempty"`
	Type              string                 `json:"type"`
	Title             string                 `json:"title,omitempty"`
	Content           string                 `json:"content"`
	Status            string                 `json:"status"`
	RefinementHistory []RefinementEntryDTO   `json:"refinement_history,omitempty"`
	PublishedAt       *string                `json:"published_at,omitempty"`
	LinkedInPostID    string                 `json:"linkedin_post_id,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         string                 `json:"created_at"`
	UpdatedAt         string                 `json:"updated_at"`
}

// RefinementEntryDTO represents a refinement entry in the response
type RefinementEntryDTO struct {
	Timestamp string `json:"timestamp"`
	Prompt    string `json:"prompt"`
	Content   string `json:"content"`
	Version   int    `json:"version"`
}

// GetDrafts handles GET /v1/drafts/{userId}
func (h *DraftsHandler) GetDrafts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract userID from path
	vars := mux.Vars(r)
	userID := vars["userId"]

	// Validate userID
	if userID == "" {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, "user_id is required", nil, h.logger)
		return
	}

	if !isValidObjectID(userID) {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, "invalid user_id format", nil, h.logger)
		return
	}

	// Parse query parameters
	queryParams := r.URL.Query()
	statusStr := queryParams.Get("status")
	typeStr := queryParams.Get("type")
	limitStr := queryParams.Get("limit")

	var limit int
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			WriteError(w, http.StatusBadRequest, ErrorCodeValidation, "invalid limit parameter", nil, h.logger)
			return
		}
		limit = parsedLimit
	}

	// Validate request
	req := ListDraftsRequest{
		Status: statusStr,
		Type:   typeStr,
		Limit:  limit,
	}

	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
		return
	}

	// Convert status and type to domain types
	var status valueobjects.DraftStatus
	if statusStr != "" {
		status = valueobjects.DraftStatus(strings.ToUpper(statusStr))
	}

	var draftType valueobjects.DraftType
	if typeStr != "" {
		draftType = valueobjects.DraftType(strings.ToUpper(typeStr))
	}

	// Query repository
	drafts, err := h.draftRepository.ListByUserID(ctx, userID, entities.DraftStatus(status), entities.DraftType(draftType))
	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	// Apply limit if specified
	if limit > 0 && len(drafts) > limit {
		drafts = drafts[:limit]
	}

	// Convert to DTOs
	draftDTOs := make([]DraftDTO, 0, len(drafts))
	for _, draft := range drafts {
		dto := DraftDTO{
			ID:             draft.ID,
			UserID:         draft.UserID,
			IdeaID:         draft.IdeaID,
			Type:           string(draft.Type),
			Title:          draft.Title,
			Content:        draft.Content,
			Status:         string(draft.Status),
			LinkedInPostID: draft.LinkedInPostID,
			Metadata:       draft.Metadata,
			CreatedAt:      draft.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      draft.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if draft.PublishedAt != nil {
			publishedAtStr := draft.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
			dto.PublishedAt = &publishedAtStr
		}

		// Convert refinement history
		if len(draft.RefinementHistory) > 0 {
			refinements := make([]RefinementEntryDTO, 0, len(draft.RefinementHistory))
			for _, entry := range draft.RefinementHistory {
				refinements = append(refinements, RefinementEntryDTO{
					Timestamp: entry.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
					Prompt:    entry.Prompt,
					Content:   entry.Content,
					Version:   entry.Version,
				})
			}
			dto.RefinementHistory = refinements
		}

		draftDTOs = append(draftDTOs, dto)
	}

	// Return response
	response := GetDraftsResponse{
		Drafts: draftDTOs,
		Count:  len(draftDTOs),
	}

	WriteJSON(w, http.StatusOK, response, h.logger)
}

// RefineDraftResponse represents the response for draft refinement
type RefineDraftResponse struct {
	Draft DraftDTO `json:"draft"`
}

// RefineDraft handles POST /v1/drafts/{draftId}/refine
func (h *DraftsHandler) RefineDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract draftId from path
	vars := mux.Vars(r)
	draftID := vars["draftId"]

	// Validate draftID
	if draftID == "" {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, "draft_id is required", nil, h.logger)
		return
	}

	if !isValidObjectID(draftID) {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, "invalid draft_id format", nil, h.logger)
		return
	}

	// Parse request body
	var req RefineDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeInvalidInput, "Invalid request body", nil, h.logger)
		return
	}
	defer r.Body.Close()

	// Validate request
	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
		return
	}

	// Execute use case
	draft, err := h.refineDraftUseCase.Execute(ctx, usecases.RefineDraftInput{
		DraftID:    draftID,
		UserPrompt: req.Prompt,
	})

	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	// Convert to DTO
	dto := DraftDTO{
		ID:             draft.ID,
		UserID:         draft.UserID,
		IdeaID:         draft.IdeaID,
		Type:           string(draft.Type),
		Title:          draft.Title,
		Content:        draft.Content,
		Status:         string(draft.Status),
		LinkedInPostID: draft.LinkedInPostID,
		Metadata:       draft.Metadata,
		CreatedAt:      draft.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      draft.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if draft.PublishedAt != nil {
		publishedAtStr := draft.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
		dto.PublishedAt = &publishedAtStr
	}

	// Convert refinement history
	if len(draft.RefinementHistory) > 0 {
		refinements := make([]RefinementEntryDTO, 0, len(draft.RefinementHistory))
		for _, entry := range draft.RefinementHistory {
			refinements = append(refinements, RefinementEntryDTO{
				Timestamp: entry.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
				Prompt:    entry.Prompt,
				Content:   entry.Content,
				Version:   entry.Version,
			})
		}
		dto.RefinementHistory = refinements
	}

	h.logger.Info("draft refined",
		zap.String("draft_id", draftID),
		zap.Int("refinements_count", len(draft.RefinementHistory)),
	)

	// Return response
	response := RefineDraftResponse{
		Draft: dto,
	}

	WriteJSON(w, http.StatusOK, response, h.logger)
}

// RegisterRoutes registers draft routes
func (h *DraftsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/v1/drafts/generate", h.GenerateDrafts).Methods(http.MethodPost)
	router.HandleFunc("/v1/drafts/{userId}", h.GetDrafts).Methods(http.MethodGet)
	router.HandleFunc("/v1/drafts/{draftId}/refine", h.RefineDraft).Methods(http.MethodPost)
}
