package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/linkgen-ai/backend/src/application/usecases"
	"go.uber.org/zap"
)

// IdeasHandler handles idea-related HTTP requests
type IdeasHandler struct {
	listIdeasUseCase  *usecases.ListIdeasUseCase
	clearIdeasUseCase *usecases.ClearIdeasUseCase
	logger            *zap.Logger
}

// NewIdeasHandler creates a new IdeasHandler instance
func NewIdeasHandler(
	listIdeasUseCase *usecases.ListIdeasUseCase,
	clearIdeasUseCase *usecases.ClearIdeasUseCase,
	logger *zap.Logger,
) *IdeasHandler {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &IdeasHandler{
		listIdeasUseCase:  listIdeasUseCase,
		clearIdeasUseCase: clearIdeasUseCase,
		logger:            logger,
	}
}

// GetIdeasResponse represents the response for listing ideas
type GetIdeasResponse struct {
	Ideas []IdeaDTO `json:"ideas"`
	Count int       `json:"count"`
}

// IdeaDTO represents an idea in the response
type IdeaDTO struct {
	ID           string   `json:"id"`
	UserID       string   `json:"user_id"`
	TopicID      string   `json:"topic_id"`
	Content      string   `json:"content"`
	QualityScore *float64 `json:"quality_score,omitempty"`
	Used         bool     `json:"used"`
	CreatedAt    string   `json:"created_at"`
	ExpiresAt    *string  `json:"expires_at,omitempty"`
}

// GetIdeas handles GET /v1/ideas/{userId}
func (h *IdeasHandler) GetIdeas(w http.ResponseWriter, r *http.Request) {
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
	topic := queryParams.Get("topic")
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
	req := ListIdeasRequest{
		Topic: topic,
		Limit: limit,
	}

	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
		return
	}

	// Execute use case
	ideas, err := h.listIdeasUseCase.Execute(ctx, usecases.ListIdeasInput{
		UserID:  userID,
		TopicID: topic,
		Limit:   limit,
	})

	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	// Convert to DTOs
	ideaDTOs := make([]IdeaDTO, 0, len(ideas))
	for _, idea := range ideas {
		dto := IdeaDTO{
			ID:           idea.ID,
			UserID:       idea.UserID,
			TopicID:      idea.TopicID,
			Content:      idea.Content,
			QualityScore: idea.QualityScore,
			Used:         idea.Used,
			CreatedAt:    idea.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if idea.ExpiresAt != nil {
			expiresAtStr := idea.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
			dto.ExpiresAt = &expiresAtStr
		}

		ideaDTOs = append(ideaDTOs, dto)
	}

	// Return response
	response := GetIdeasResponse{
		Ideas: ideaDTOs,
		Count: len(ideaDTOs),
	}

	WriteJSON(w, http.StatusOK, response, h.logger)
}

// ClearIdeas handles DELETE /v1/ideas/{userId}/clear
func (h *IdeasHandler) ClearIdeas(w http.ResponseWriter, r *http.Request) {
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

	// Execute use case
	result, err := h.clearIdeasUseCase.Execute(ctx, usecases.ClearIdeasInput{
		UserID: userID,
	})

	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	// Return 204 No Content on success
	h.logger.Info("ideas cleared",
		zap.String("user_id", userID),
		zap.Int64("deleted_count", result.DeletedCount),
	)

	w.WriteHeader(http.StatusNoContent)
}

// ClearIdeasResponse represents the response for clearing ideas (for debugging)
type ClearIdeasResponse struct {
	DeletedCount int64  `json:"deleted_count"`
	Message      string `json:"message"`
}

// RegisterRoutes registers idea routes
func (h *IdeasHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/v1/ideas/{userId}", h.GetIdeas).Methods(http.MethodGet)
	router.HandleFunc("/v1/ideas/{userId}/clear", h.ClearIdeas).Methods(http.MethodDelete)
}
