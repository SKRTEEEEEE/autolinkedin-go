package services

import (
	"context"
	"fmt"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

const (
	// DevUserID is the fixed user ID for local development
	// Using a valid MongoDB ObjectID format (24 hex characters)
	DevUserID = "000000000000000000000001"
)

// DevSeeder handles seeding development data
type DevSeeder struct {
	userRepo  interfaces.UserRepository
	topicRepo interfaces.TopicRepository
	logger    *zap.Logger
}

// NewDevSeeder creates a new development data seeder
func NewDevSeeder(
	userRepo interfaces.UserRepository,
	topicRepo interfaces.TopicRepository,
	logger *zap.Logger,
) *DevSeeder {
	return &DevSeeder{
		userRepo:  userRepo,
		topicRepo: topicRepo,
		logger:    logger,
	}
}

// SeedDevUser creates or updates the development user
func (s *DevSeeder) SeedDevUser(ctx context.Context) error {
	s.logger.Info("Seeding development user...")

	// Check if dev user already exists
	existingUser, err := s.userRepo.FindByID(ctx, DevUserID)
	if err == nil && existingUser != nil {
		s.logger.Info("Development user already exists", zap.String("user_id", DevUserID))
		return nil
	}

	// Create new dev user
	user := &entities.User{
		ID:            DevUserID,
		Email:         "dev@local.linkgen.ai",
		LinkedInToken: "dev-token-not-needed-for-local",
		APIKeys:       map[string]string{"dev": "key"},
		Configuration: map[string]interface{}{
			"auto_publish": false,
			"environment":  "development",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	// Validate user
	if err := user.Validate(); err != nil {
		return fmt.Errorf("failed to validate dev user: %w", err)
	}

	// Save user
	userID, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to save dev user: %w", err)
	}

	s.logger.Info("Development user created successfully", zap.String("user_id", userID))
	return nil
}

// SeedDefaultTopics creates default topics for the dev user if they don't exist
func (s *DevSeeder) SeedDefaultTopics(ctx context.Context) error {
	s.logger.Info("Seeding default topics for development user...")

	defaultTopics := []struct {
		name        string
		description string
	}{
		{
			name:        "Technology",
			description: "Software development, AI, cloud computing, and tech innovations",
		},
		{
			name:        "Productivity",
			description: "Time management, work efficiency, and productivity hacks",
		},
		{
			name:        "Leadership",
			description: "Team management, leadership skills, and organizational culture",
		},
	}

	for _, topicData := range defaultTopics {
		// Check if topic already exists for this user
		topics, err := s.topicRepo.ListByUserID(ctx, DevUserID)
		if err != nil {
			s.logger.Warn("Failed to check existing topics", zap.Error(err))
		}

		// Check if this topic name already exists
		topicExists := false
		for _, t := range topics {
			if t.Name == topicData.name {
				topicExists = true
				break
			}
		}

		if topicExists {
			s.logger.Info("Topic already exists, skipping", zap.String("topic", topicData.name))
			continue
		}

		// Create new topic
		topic := &entities.Topic{
			ID:          primitive.NewObjectID().Hex(),
			UserID:      DevUserID,
			Name:        topicData.name,
			Description: topicData.description,
			Keywords:    []string{},
			Category:    "General",
			Priority:    5,
			Active:      true,
			CreatedAt:   time.Now(),
		}

		// Validate topic
		if err := topic.Validate(); err != nil {
			s.logger.Warn("Failed to validate topic",
				zap.String("topic", topicData.name),
				zap.Error(err),
			)
			continue
		}

		// Save topic
		topicID, err := s.topicRepo.Create(ctx, topic)
		if err != nil {
			s.logger.Warn("Failed to save topic",
				zap.String("topic", topicData.name),
				zap.Error(err),
			)
			continue
		}

		s.logger.Info("Default topic created",
			zap.String("topic", topicData.name),
			zap.String("topic_id", topicID),
		)
	}

	s.logger.Info("Default topics seeding completed")
	return nil
}

// SeedAll seeds all development data
func (s *DevSeeder) SeedAll(ctx context.Context) error {
	if err := s.SeedDevUser(ctx); err != nil {
		return fmt.Errorf("failed to seed dev user: %w", err)
	}

	if err := s.SeedDefaultTopics(ctx); err != nil {
		return fmt.Errorf("failed to seed default topics: %w", err)
	}

	return nil
}
