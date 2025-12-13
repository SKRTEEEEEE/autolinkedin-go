package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	promptsRepo     interfaces.PromptsRepository
	generateIdeasUC GenerateIdeasUseCase
	logger          *zap.Logger
}

// GenerateIdeasUseCase defines the interface for generating ideas
type GenerateIdeasUseCase interface {
	GenerateIdeasForUser(ctx context.Context, userID string, count int) ([]*entities.Idea, error)
	GenerateIdeasForTopic(ctx context.Context, topicID string) ([]*entities.Idea, error)
}

// NewTopicsHandler creates a new TopicsHandler instance
func NewTopicsHandler(
	topicRepo interfaces.TopicRepository,
	userRepo interfaces.UserRepository,
	promptsRepo interfaces.PromptsRepository,
	generateIdeasUC GenerateIdeasUseCase,
	logger *zap.Logger,
) *TopicsHandler {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &TopicsHandler{
		topicRepo:       topicRepo,
		userRepo:        userRepo,
		promptsRepo:     promptsRepo,
		generateIdeasUC: generateIdeasUC,
		logger:          logger,
	}
}

// TopicDTO represents a topic in the response
type TopicDTO struct {
	ID            string   `json:"id"`
	UserID        string   `json:"user_id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Category      string   `json:"category,omitempty"`
	Priority      int      `json:"priority"`
	Ideas         int      `json:"ideas"`
	Prompt        string   `json:"prompt"`
	RelatedTopics []string `json:"related_topics,omitempty"`
	Active        bool     `json:"active"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// CreateTopicRequest represents the request to create a topic
type CreateTopicRequest struct {
	UserID        string   `json:"user_id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Category      string   `json:"category,omitempty"`
	Priority      *int     `json:"priority,omitempty"`
	Ideas         *int     `json:"ideas,omitempty"`
	Prompt        *string  `json:"prompt,omitempty"`
	RelatedTopics []string `json:"related_topics,omitempty"`
	Active        *bool    `json:"active,omitempty"`
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
	if len(r.Category) > 50 {
		return fmt.Errorf("category must be less than 50 characters")
	}
	if r.Priority != nil && (*r.Priority < 1 || *r.Priority > 10) {
		return fmt.Errorf("priority must be between 1 and 10")
	}
	if r.Ideas != nil && (*r.Ideas < 1 || *r.Ideas > 20) {
		return fmt.Errorf("ideas must be between 1 and 20")
	}
	if r.Prompt != nil && len(*r.Prompt) > 50 {
		return fmt.Errorf("prompt must be less than 50 characters")
	}
	if len(r.RelatedTopics) > 10 {
		return fmt.Errorf("related_topics must contain 10 or fewer items")
	}
	return nil
}

// UpdateTopicRequest represents the request to update a topic
type UpdateTopicRequest struct {
	Name          string   `json:"name,omitempty"`
	Description   string   `json:"description,omitempty"`
	Category      string   `json:"category,omitempty"`
	Priority      *int     `json:"priority,omitempty"`
	Ideas         *int     `json:"ideas,omitempty"`
	Prompt        *string  `json:"prompt,omitempty"`
	RelatedTopics []string `json:"related_topics,omitempty"`
	Active        *bool    `json:"active,omitempty"`
}

// Validate validates the update topic request
func (r *UpdateTopicRequest) Validate() error {
	if r.Name != "" && len(r.Name) > 100 {
		return fmt.Errorf("name must be less than 100 characters")
	}
	if len(r.Description) > 500 {
		return fmt.Errorf("description must be less than 500 characters")
	}
	if len(r.Category) > 50 {
		return fmt.Errorf("category must be less than 50 characters")
	}
	if r.Priority != nil && (*r.Priority < 1 || *r.Priority > 10) {
		return fmt.Errorf("priority must be between 1 and 10")
	}
	if r.Ideas != nil && (*r.Ideas < 1 || *r.Ideas > 20) {
		return fmt.Errorf("ideas must be between 1 and 20")
	}
	if r.Prompt != nil && len(*r.Prompt) > 50 {
		return fmt.Errorf("prompt must be less than 50 characters")
	}
	if len(r.RelatedTopics) > 10 {
		return fmt.Errorf("related_topics must contain 10 or fewer items")
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
			ID:            topic.ID,
			UserID:        topic.UserID,
			Name:          topic.Name,
			Description:   topic.Description,
			Category:      topic.Category,
			Priority:      topic.Priority,
			Ideas:         topic.Ideas,
			Prompt:        topic.Prompt,
			RelatedTopics: topic.RelatedTopics,
			Active:        topic.Active,
			CreatedAt:     topic.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:     topic.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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

	if err := h.validatePromptReference(ctx, req.UserID, req.Prompt); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
		return
	}

	// Create topic entity
	topic := &entities.Topic{
		ID:            primitive.NewObjectID().Hex(),
		UserID:        req.UserID,
		Name:          req.Name,
		Description:   req.Description,
		Category:      req.Category,
		Priority:      entities.DefaultPriority,
		Ideas:         entities.DefaultIdeasCount,
		RelatedTopics: req.RelatedTopics,
		Active:        true,
		CreatedAt:     time.Now(),
	}

	// Set optional fields if provided
	if req.Category != "" {
		topic.Category = req.Category
	}
	if req.Priority != nil {
		topic.Priority = *req.Priority
	}
	if req.Ideas != nil {
		topic.Ideas = *req.Ideas
	}
	if req.Prompt != nil {
		topic.Prompt = strings.TrimSpace(*req.Prompt)
	}
	if req.Active != nil {
		topic.Active = *req.Active
	}

	// Set defaults and validate
	topic.SetDefaults()
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

	// Use the specific topic's prompt for idea generation instead of auto-generating
	if h.generateIdeasUC != nil {
		go func() {
			generateCtx := context.Background()
			_, generateErr := h.generateIdeasUC.GenerateIdeasForTopic(generateCtx, topicID)
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
		ID:            topicID,
		UserID:        topic.UserID,
		Name:          topic.Name,
		Description:   topic.Description,
		Category:      topic.Category,
		Priority:      topic.Priority,
		Ideas:         topic.Ideas,
		Prompt:        topic.Prompt,
		RelatedTopics: topic.RelatedTopics,
		Active:        topic.Active,
		CreatedAt:     topic.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     topic.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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
	if req.Category != "" {
		topic.Category = req.Category
	}
	if req.Priority != nil {
		topic.Priority = *req.Priority
	}
	if req.Ideas != nil {
		topic.Ideas = *req.Ideas
	}
	if req.Prompt != nil {
		topic.Prompt = strings.TrimSpace(*req.Prompt)
	}
	if req.RelatedTopics != nil {
		topic.RelatedTopics = req.RelatedTopics
		topic.NormalizeRelatedTopics()
	}
	if req.Active != nil {
		topic.Active = *req.Active
	}

	if req.Prompt != nil {
		if err := h.validatePromptReference(ctx, topic.UserID, req.Prompt); err != nil {
			WriteError(w, http.StatusBadRequest, ErrorCodeValidation, err.Error(), nil, h.logger)
			return
		}
	}

	topic.UpdatedAt = time.Now()

	// Set defaults and validate
	topic.SetDefaults()
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
		ID:            topic.ID,
		UserID:        topic.UserID,
		Name:          topic.Name,
		Description:   topic.Description,
		Category:      topic.Category,
		Priority:      topic.Priority,
		Ideas:         topic.Ideas,
		Prompt:        topic.Prompt,
		RelatedTopics: topic.RelatedTopics,
		Active:        topic.Active,
		CreatedAt:     topic.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     topic.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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

func (h *TopicsHandler) validatePromptReference(ctx context.Context, userID string, promptName *string) error {
	if promptName == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*promptName)
	if trimmed == "" {
		return fmt.Errorf("prompt reference cannot be empty")
	}

	prompt, err := h.promptsRepo.FindByName(ctx, userID, trimmed)
	if err != nil {
		return fmt.Errorf("failed to validate prompt reference: %w", err)
	}

	if prompt == nil {
		return fmt.Errorf("prompt reference not found: %s", trimmed)
	}

	if prompt.Type != entities.PromptTypeIdeas {
		return fmt.Errorf("prompt reference must be of ideas type")
	}

	return nil
}

// RegisterRoutes registers all topic routes
func (h *TopicsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/v1/topics/{userId}", h.GetTopics).Methods(http.MethodGet)
	router.HandleFunc("/v1/topics", h.CreateTopic).Methods(http.MethodPost)
	router.HandleFunc("/v1/topics/{topicId}", h.UpdateTopic).Methods(http.MethodPut)
	router.HandleFunc("/v1/topics/{topicId}", h.DeleteTopic).Methods(http.MethodDelete)
}
