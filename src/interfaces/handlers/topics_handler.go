package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"context"

	"github.com/gorilla/mux"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// TopicsHandler handles topic-related HTTP requests
type TopicsHandler struct {
	topicRepo       interfaces.TopicRepository
	userRepo        interfaces.UserRepository
	generateIdeasUC GenerateIdeasUseCase
	logger          *zap.Logger
}

// GenerateIdeasUseCase defines the interface for generating ideas
type GenerateIdeasUseCase interface {
	GenerateIdeasForUser(ctx context.Context, userID string, count int) ([]*entities.Idea, error)
}

// NewTopicsHandler creates a new TopicsHandler instance
func NewTopicsHandler(
	topicRepo interfaces.TopicRepository,
	userRepo interfaces.UserRepository,
	generateIdeasUC GenerateIdeasUseCase,
	logger *zap.Logger,
) *TopicsHandler {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &TopicsHandler{
		topicRepo:       topicRepo,
		userRepo:        userRepo,
		generateIdeasUC: generateIdeasUC,
		logger:          logger,
	}
}

// TopicDTO represents a topic in the response
type TopicDTO struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Category    string   `json:"category,omitempty"`
	Priority    int      `json:"priority"`
	Active      bool     `json:"active"`
	CreatedAt   string   `json:"created_at"`
}

// CreateTopicRequest represents the request to create a topic
type CreateTopicRequest struct {
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Validate validates the create topic request
func (r *CreateTopicRequest) Validate() error {
	if r.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) > 100 {
		return fmt.Errorf("name must be less than 100 characters")
	}
	if len(r.Description) > 500 {
		return fmt.Errorf("description must be less than 500 characters")
	}
	return nil
}

// UpdateTopicRequest represents the request to update a topic
type UpdateTopicRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Category    string   `json:"category,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	Active      *bool    `json:"active,omitempty"`
}

// Validate validates the update topic request
func (r *UpdateTopicRequest) Validate() error {
	if r.Name != "" && len(r.Name) > 100 {
		return fmt.Errorf("name must be less than 100 characters")
	}
	if len(r.Description) > 500 {
		return fmt.Errorf("description must be less than 500 characters")
	}
	return nil
}

// GetTopics handles GET /v1/topics/{userId}
func (h *TopicsHandler) GetTopics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract userID from path
	vars := mux.Vars(r)
	userID := vars["userId"]

	// Validate userID
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

	// Get topics
	topics, err := h.topicRepo.ListByUserID(ctx, userID)
	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	// Convert to DTOs
	topicDTOs := make([]TopicDTO, 0, len(topics))
	for _, topic := range topics {
		topicDTOs = append(topicDTOs, TopicDTO{
			ID:          topic.ID,
			UserID:      topic.UserID,
			Name:        topic.Name,
			Description: topic.Description,
			Keywords:    topic.Keywords,
			Category:    topic.Category,
			Priority:    topic.Priority,
			Active:      topic.Active,
			CreatedAt:   topic.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// Return response
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"topics": topicDTOs,
		"count":  len(topicDTOs),
	}, h.logger)
}

// CreateTopic handles POST /v1/topics
func (h *TopicsHandler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req CreateTopicRequest
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

	// Create topic entity
	topic := &entities.Topic{
		ID:          primitive.NewObjectID().Hex(),
		UserID:      req.UserID,
		Name:        req.Name,
		Description: req.Description,
		Keywords:    []string{},
		Category:    "General",
		Priority:    5,
		Active:      true,
		CreatedAt:   time.Now(),
	}

	// Validate topic
	if err := topic.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
		return
	}

	// Save topic
	topicID, err := h.topicRepo.Create(ctx, topic)
	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	// Trigger idea generation for new topic (async, don't block response)
	if h.generateIdeasUC != nil {
		go func() {
			generateCtx := context.Background()
			_, generateErr := h.generateIdeasUC.GenerateIdeasForUser(generateCtx, req.UserID, 10)
			if generateErr != nil {
				h.logger.Warn("Failed to auto-generate ideas for new topic",
					zap.String("topic_id", topicID),
					zap.Error(generateErr))
			} else {
				h.logger.Info("Auto-generated ideas for new topic",
					zap.String("topic_id", topicID))
			}
		}()
	}

	// Return created topic
	WriteJSON(w, http.StatusCreated, TopicDTO{
		ID:          topicID,
		UserID:      topic.UserID,
		Name:        topic.Name,
		Description: topic.Description,
		Keywords:    topic.Keywords,
		Category:    topic.Category,
		Priority:    topic.Priority,
		Active:      topic.Active,
		CreatedAt:   topic.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, h.logger)
}

// UpdateTopic handles PUT /v1/topics/{topicId}
func (h *TopicsHandler) UpdateTopic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract topicID from path
	vars := mux.Vars(r)
	topicID := vars["topicId"]

	if topicID == "" {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, "topic_id is required", nil, h.logger)
		return
	}

	// Parse request body
	var req UpdateTopicRequest
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

	// Get existing topic
	topic, err := h.topicRepo.FindByID(ctx, topicID)
	if err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}
	if topic == nil {
		WriteError(w, http.StatusNotFound, ErrorCodeNotFound, "topic not found", nil, h.logger)
		return
	}

	// Update fields
	if req.Name != "" {
		topic.Name = req.Name
	}
	if req.Description != "" {
		topic.Description = req.Description
	}
	if req.Keywords != nil {
		topic.Keywords = req.Keywords
	}
	if req.Category != "" {
		topic.Category = req.Category
	}
	if req.Priority != nil {
		topic.Priority = *req.Priority
	}
	if req.Active != nil {
		topic.Active = *req.Active
	}

	// Validate updated topic
	if err := topic.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
		return
	}

	// Persist changes
	if err := h.topicRepo.Update(ctx, topic); err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	// Return updated topic
	WriteJSON(w, http.StatusOK, TopicDTO{
		ID:          topic.ID,
		UserID:      topic.UserID,
		Name:        topic.Name,
		Description: topic.Description,
		Keywords:    topic.Keywords,
		Category:    topic.Category,
		Priority:    topic.Priority,
		Active:      topic.Active,
		CreatedAt:   topic.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, h.logger)
}

// DeleteTopic handles DELETE /v1/topics/{topicId}
func (h *TopicsHandler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract topicID from path
	vars := mux.Vars(r)
	topicID := vars["topicId"]

	if topicID == "" {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, "topic_id is required", nil, h.logger)
		return
	}

	// Delete topic
	if err := h.topicRepo.Delete(ctx, topicID); err != nil {
		statusCode, code, message := MapDomainError(err, h.logger)
		WriteError(w, statusCode, code, message, nil, h.logger)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// RegisterRoutes registers all topic routes
func (h *TopicsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/v1/topics/{userId}", h.GetTopics).Methods(http.MethodGet)
	router.HandleFunc("/v1/topics", h.CreateTopic).Methods(http.MethodPost)
	router.HandleFunc("/v1/topics/{topicId}", h.UpdateTopic).Methods(http.MethodPut)
	router.HandleFunc("/v1/topics/{topicId}", h.DeleteTopic).Methods(http.MethodDelete)
}
